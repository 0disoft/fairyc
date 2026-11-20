package sourceedit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type batchStage struct {
	path string
	info os.FileInfo
}

type batchPublication struct {
	batchFile
	original    os.FileInfo
	replacement batchStage
	backup      batchStage
}

// ApplyWithOptions stages both versions before replacing any source. A
// failed rename rolls back the completed prefix in reverse order.
func (p *BatchPlan) ApplyWithOptions(options ...ApplyOptions) (err error) {
	policy, err := editApplyPolicy(options)
	if err != nil {
		return err
	}
	ctx := policy.Context
	if p == nil || len(p.files) == 0 || p.baseline == nil {
		return fmt.Errorf("invalid or already applied edit batch")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	root, err := os.OpenRoot(p.root)
	if err != nil {
		return err
	}
	defer root.Close()
	if err := root.Mkdir(editLock, 0700); err != nil {
		return fmt.Errorf("exclusive edit lock unavailable: %w", err)
	}

	var journal batchStage
	publications := make([]batchPublication, 0, len(p.files))
	keepRecovery := false
	defer func() {
		for _, file := range publications {
			err = errors.Join(err, file.replacement.remove(root))
			if !keepRecovery {
				err = errors.Join(err, file.backup.remove(root))
			}
		}
		if !keepRecovery {
			err = errors.Join(err, journal.remove(root), root.Remove(editLock))
		}
		if err != nil && p.baseline == nil {
			err = fmt.Errorf("edit batch published before cleanup failed: %w", err)
		}
	}()

	if err := p.checkBatchBaseline(ctx); err != nil {
		return err
	}
	overlays := make(map[string][]byte, len(p.files))
	for _, file := range p.files {
		overlays[filepath.Join(p.root, filepath.FromSlash(file.path))] = file.after
	}
	if err := validateOverlays(ctx, p.root, overlays); err != nil {
		return err
	}
	for _, file := range p.files {
		publications = append(publications, batchPublication{batchFile: file})
		if err := publications[len(publications)-1].stage(root, policy); err != nil {
			return fmt.Errorf("stage batch source %s: %w", file.path, err)
		}
	}
	journal, err = writeBatchJournal(root, publications, policy)
	if err != nil {
		return err
	}
	if err := p.checkBatchBaseline(ctx); err != nil {
		return err
	}
	for _, file := range publications {
		if err := verifyEditStage(ctx, root, file.backup.path, file.backup.info, file.before); err != nil {
			return fmt.Errorf("verify backup for %s: %w", file.path, err)
		}
	}
	for i, file := range publications {
		if err := file.publish(root, policy); err != nil {
			rollbackErr := rollbackBatch(context.WithoutCancel(ctx), root, publications[:i], policy)
			keepRecovery = rollbackErr != nil
			if keepRecovery {
				rollbackErr = fmt.Errorf("batch recovery required; inspect %s: %w", filepath.ToSlash(journal.path), rollbackErr)
			}
			return errors.Join(fmt.Errorf("publish batch source %s: %w", file.path, err), rollbackErr)
		}
		publications[i].replacement.path = "" // The destination now owns the staged object.
	}
	for i := range p.Report.Edits {
		p.Report.Edits[i].Applied = true
	}
	p.baseline = nil
	return nil
}

func (p *BatchPlan) checkBatchBaseline(ctx context.Context) error {
	current, err := p.baseline.Matches(ctx, nil)
	if err != nil {
		return err
	}
	if !current {
		return fmt.Errorf("workspace changed since edit batch preview")
	}
	return nil
}

func (file *batchPublication) stage(root *os.Root, policy ApplyOptions) error {
	path := filepath.FromSlash(file.path)
	info, err := root.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0200 == 0 {
		return fmt.Errorf("source must be a writable regular file")
	}
	file.original = info
	directory, mode := filepath.Dir(path), info.Mode().Perm()
	file.replacement, err = stageBatchFile(root, directory, file.after, mode, policy)
	if err != nil {
		return err
	}
	file.backup, err = stageBatchFile(root, directory, file.before, mode, policy)
	return err
}

func stageBatchFile(root *os.Root, directory string, data []byte, mode os.FileMode, policy ApplyOptions) (batchStage, error) {
	f, name, err := policy.CreateTemporary(root, directory)
	if err != nil {
		if f != nil {
			err = errors.Join(err, f.Close())
		}
		return batchStage{}, err
	}
	if f == nil {
		return batchStage{}, fmt.Errorf("edit staging file is missing")
	}
	info, err := f.Stat()
	if err != nil {
		return batchStage{}, errors.Join(err, f.Close())
	}
	staged := batchStage{name, info}
	return staged, writeEditStage(f, data, mode, policy)
}

func (stage batchStage) remove(root *os.Root) error {
	if stage.path == "" {
		return nil
	}
	return removeEditStage(root, stage.path, stage.info)
}

func (file batchPublication) publish(root *os.Root, policy ApplyOptions) error {
	path := filepath.FromSlash(file.path)
	if err := verifyEditStage(policy.Context, root, path, file.original, file.before); err != nil {
		return err
	}
	if err := verifyEditStage(policy.Context, root, file.replacement.path, file.replacement.info, file.after); err != nil {
		return err
	}
	return policy.Rename(root, file.replacement.path, path)
}

func rollbackBatch(ctx context.Context, root *os.Root, published []batchPublication, policy ApplyOptions) (result error) {
	for i := len(published) - 1; i >= 0; i-- {
		file := &published[i]
		path := filepath.FromSlash(file.path)
		err := verifyEditStage(ctx, root, path, file.replacement.info, file.after)
		if err == nil {
			err = verifyEditStage(ctx, root, file.backup.path, file.backup.info, file.before)
		}
		if err == nil {
			err = policy.Rename(root, file.backup.path, path)
		}
		if err != nil {
			result = errors.Join(result, fmt.Errorf("restore %s: %w", file.path, err))
		} else {
			file.backup.path = ""
		}
	}
	return result
}

func writeBatchJournal(root *os.Root, files []batchPublication, policy ApplyOptions) (batchStage, error) {
	type recoveryFile struct {
		Path         string `json:"path"`
		Backup       string `json:"backup"`
		BeforeSHA256 string `json:"beforeSha256"`
		AfterSHA256  string `json:"afterSha256"`
	}
	record := struct {
		Schema string         `json:"schema"`
		Files  []recoveryFile `json:"files"`
	}{Schema: "fairyc.edit-batch-recovery/v1", Files: make([]recoveryFile, 0, len(files))}
	for _, file := range files {
		record.Files = append(record.Files, recoveryFile{
			file.path, filepath.ToSlash(file.backup.path), Digest(file.before), Digest(file.after),
		})
	}
	data, err := json.Marshal(record)
	if err != nil {
		return batchStage{}, err
	}
	return stageBatchFile(root, editLock, data, 0600, policy)
}

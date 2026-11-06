package contractstatenative

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/target"
)

const (
	graphInputHeader  = 152
	graphOutputHeader = 186
	graphEventHeader  = 21
)

// GraphTransport admits a path witness from one private compiled graph owner.
// It grants no storage authority and never executes MIR to manufacture outputs.
type GraphTransport struct {
	entry   GraphArtifact
	program [32]byte
	sites   map[uint32]graphSite
	start   uint32
	input   OverlayTransport
	work    uint64
	literal int
}

type GraphInput struct {
	packet []byte
	valid  bool
}

func (input GraphInput) Bytes() []byte { return slices.Clone(input.packet) }

type GraphResult struct {
	Status       byte
	Exit         contractstatemir.ExitKind
	TrapCategory string
	Version      uint64
	Cost         uint64
	Operations   uint64
	Reads        []contractstatemir.Read
	Writes       []contractstate.PersistentWrite
	WriteBytes   uint64
}

func PrepareGraphTransport(handoff Handoff, id target.ID) (GraphTransport, error) {
	entry, err := emitGraph(handoff, id, true)
	if err != nil {
		return GraphTransport{}, err
	}
	prepared := GraphTransport{entry: entry}
	prepared.sites, prepared.start = graphPlan(handoff.program, entry.traps)
	prepared.work, prepared.literal = graphWorkBounds(handoff.program)
	identity := append([]byte("fairyc.native-state-graph-transport/v1\x00"), entry.program[:]...)
	identity = binary.BigEndian.AppendUint64(identity, prepared.work)
	identity = binary.BigEndian.AppendUint64(identity, uint64(prepared.literal))
	prepared.program = sha256.Sum256(append(identity, graphNativeFrameC...))
	// Reuse snapshot framing/validation, not the literal operation or write plan.
	prepared.input = OverlayTransport{
		entry: OverlayArtifact{layout: entry.layout, body: entry.body, programHash: prepared.program, valid: true},
		known: graphKnownCells(handoff.program.Layout.Fields),
	}
	return prepared, nil
}

func (prepared GraphTransport) Input(snapshot contractstate.PersistentSnapshot, limits contractstate.Limits) (GraphInput, error) {
	if !prepared.entry.valid || prepared.start == 0 {
		return GraphInput{}, fmt.Errorf("graph request requires a prepared owner")
	}
	input, err := prepared.input.Input(snapshot, limits)
	if err != nil {
		return GraphInput{}, err
	}
	copy(input.packet[:8], "FYGRIN01")
	return GraphInput{packet: input.packet, valid: true}, nil
}

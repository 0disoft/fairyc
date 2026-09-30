use zed_extension_api::{self as zed, settings::LspSettings, Result};

struct FairyC;

impl zed::Extension for FairyC {
    fn new() -> Self {
        Self
    }

    fn language_server_command(
        &mut self,
        _id: &zed::LanguageServerId,
        worktree: &zed::Worktree,
    ) -> Result<zed::Command> {
        let configured = LspSettings::for_worktree("fairyc-lsp", worktree)
            .ok()
            .and_then(|settings| settings.binary)
            .and_then(|binary| binary.path);
        let command = configured
            .or_else(|| worktree.which("fairyc-lsp"))
            .ok_or_else(|| {
                "Set lsp.fairyc-lsp.binary.path to the fairyc-lsp executable".to_string()
            })?;
        Ok(zed::Command {
            command,
            args: Vec::new(),
            env: worktree.shell_env(),
        })
    }
}

zed::register_extension!(FairyC);

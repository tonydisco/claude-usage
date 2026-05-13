# oh-my-zsh plugin — claude-usage

A drop-in plugin that adds tab-completion and a cached prompt segment for the
[`claude-usage`](https://github.com/tonydisco/claude-usage) CLI.

## Install

Symlink this folder into your oh-my-zsh custom plugins directory, then enable it:

```bash
git clone https://github.com/tonydisco/claude-usage.git ~/src/claude-usage
ln -s ~/src/claude-usage/plugins/oh-my-zsh/claude-usage \
      "${ZSH_CUSTOM:-$HOME/.oh-my-zsh/custom}/plugins/claude-usage"

# Then add it to plugins=(...) in ~/.zshrc, e.g.
#   plugins=(git claude-usage)

omz reload
```

## Use

Add a prompt segment to your right-hand prompt:

```bash
# in ~/.zshrc
RPROMPT='$(claude_usage_prompt_info)'
```

You'll see `[16%/20%]` updating at most once per minute.

## Knobs

- `CLAUDE_USAGE_PROMPT_TTL` — cache lifetime in seconds (default 60).
  Set higher if you want the prompt segment to refresh less often.

The plugin is silent when:
- the `claude-usage` binary is not on `$PATH`, or
- no credential is configured / the network is unreachable.

That keeps your shell from breaking when claude.ai is down.

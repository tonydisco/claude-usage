# claude-usage oh-my-zsh plugin
#
# Install:
#   ln -s <repo>/plugins/oh-my-zsh/claude-usage "${ZSH_CUSTOM:-$ZSH/custom}/plugins/claude-usage"
#   then add `claude-usage` to plugins=(...) in ~/.zshrc
#
# Provides:
#   - shell completion (via `claude-usage completion zsh`)
#   - $(claude_usage_prompt_info) for embedding in PS1/RPROMPT
#
# Example:
#   RPROMPT='$(claude_usage_prompt_info)'

# Bail out quietly if the binary is missing — keep prompt fast.
if ! command -v claude-usage >/dev/null 2>&1; then
  return
fi

# Lazy-load completion.
if [[ -z "${_CLAUDE_USAGE_COMPLETION_LOADED:-}" ]]; then
  source <(claude-usage completion zsh)
  compdef _claude-usage claude-usage
  export _CLAUDE_USAGE_COMPLETION_LOADED=1
fi

# Cached prompt info — refreshes at most every CLAUDE_USAGE_PROMPT_TTL
# seconds (default 60) so PS1 stays snappy.
: ${CLAUDE_USAGE_PROMPT_TTL:=60}

claude_usage_prompt_info() {
  local cache_file="${TMPDIR:-/tmp}/claude-usage-prompt.$UID"
  local now=$(date +%s)
  local mtime=0
  if [[ -f "$cache_file" ]]; then
    mtime=$(stat -f %m "$cache_file" 2>/dev/null || stat -c %Y "$cache_file" 2>/dev/null || echo 0)
  fi
  if (( now - mtime >= CLAUDE_USAGE_PROMPT_TTL )); then
    claude-usage prompt --no-color 2>/dev/null >"$cache_file" &!
  fi
  [[ -s "$cache_file" ]] && cat "$cache_file"
}

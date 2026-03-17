# jterrazz shell configuration
# This file is sourced by ~/.zshrc

# jterrazz CLI binary
export PATH="$HOME/.jterrazz/bin:$PATH"

# Bun global binaries
export PATH="$HOME/.bun/bin:$PATH"

# Start interactive shells in ~/Developer when opened from HOME.
if [[ -o interactive && "$PWD" == "$HOME" && -d "$HOME/Developer" ]]; then
    cd "$HOME/Developer"
fi

# Load j command completions
if command -v j &> /dev/null; then
    eval "$(j completion zsh)"
fi

# Tmux launcher commands
jj() {
    if ! command -v tmux &>/dev/null; then
        echo "tmux not found"
        return 1
    fi

    if ! tmux has-session -t main 2>/dev/null; then
        tmux new-session -ds main
    fi

    if [[ -n "$TMUX" ]]; then
        tmux switch-client -t main
    else
        tmux attach-session -t main
    fi
}

_tmux_tool() {
    local window_name="$1"
    local tool_cmd="$2"

    if ! command -v tmux &>/dev/null; then
        echo "tmux not found"
        return 1
    fi
    if ! command -v "$tool_cmd" &>/dev/null; then
        echo "$tool_cmd not found"
        return 1
    fi

    if [[ -n "$TMUX" ]]; then
        if tmux has-session -t main 2>/dev/null; then
            tmux new-window -t main -n "$window_name" "$tool_cmd"
            tmux switch-client -t main
        else
            tmux new-session -ds main -n "$window_name" "$tool_cmd"
            tmux switch-client -t main
        fi
        return
    fi

    if tmux has-session -t main 2>/dev/null; then
        tmux new-session -t main \; set destroy-unattached on \; new-window -n "$window_name" "$tool_cmd"
        return
    fi

    tmux new-session -s main -n "$window_name" "$tool_cmd"
}

jc() { _tmux_tool "claude" "claude"; }
jo() { _tmux_tool "codex" "codex"; }
jg() { _tmux_tool "gemini" "gemini"; }

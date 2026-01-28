# ccctsl

Claude Code Cache Timer Statusline - experiments into how Claude Code's statusline feature works.

## TL;DR

**A self-updating statusline is not possible at this time.** Claude Code buffers all subprocess output and only displays it after the process exits. The statusline only updates when conversation state changes (new messages).

## How Claude Code Statuslines Actually Work

### Process Lifecycle

1. Claude Code spawns a **new process** each time it wants a statusline update
2. It passes session context as **JSON via stdin**
3. It **waits for the process to exit** before displaying output
4. Old processes are **not killed** - they accumulate if long-running (bug?)
5. Updates occur at most every 300ms, but only on conversation events

### Key Findings

| What we tested | Result |
|----------------|--------|
| Long-running process with continuous output | Output buffered until exit, then all lines displayed at once |
| `\r` carriage returns for in-place updates | Not interpreted; lines stack |
| `syscall.Write()` to bypass Go buffering | Still buffered by Claude Code |
| `stdbuf -oL` for line buffering | Still buffered by Claude Code |
| Multiple processes running | Old processes keep running, not killed |

### The Buffering is on Claude Code's Side

We confirmed this by:
1. Using `syscall.Write(1, ...)` which bypasses all Go/libc buffering
2. Output still appeared all at once after 30 seconds when the process exited
3. Therefore Claude Code itself buffers subprocess stdout until process termination

## What Works

A simple, fast-exiting command that outputs one line:

```bash
# GNU date (Linux, nix)
cat >/dev/null; echo "cache expires $(date -d '+4 minutes 40 seconds' +%H:%M:%S)"

# BSD date (macOS default)
cat >/dev/null; echo "cache expires $(date -v+4M -v+40S +%H:%M:%S)"
```

This gives you an absolute timestamp you can compare against your clock. The `cat >/dev/null` drains stdin so the shell doesn't hang.

## JSON Input Structure

Claude Code passes this to your command via stdin:

```json
{
  "model": {"display_name": "Opus 4.5"},
  "context_window": {
    "used_percentage": 42.5,
    "current_usage": {
      "cache_creation_input_tokens": 5000,
      "cache_read_input_tokens": 2000
    }
  },
  "workspace": {"current_dir": "/path/to/project"},
  "cost": {"total_cost_usd": 0.01234}
}
```

## Files in This Repo

- `ccctsl.go` - Go program we used for testing (reads JSON, outputs statusline)
- `statusline.sh` - Wrapper script for debugging

## Configuration

In `~/.claude/settings.json`:

```json
{
  "statusLine": {
    "type": "command",
    "command": "cat >/dev/null; echo \"cache expires $(date -d '+4 minutes 40 seconds' +%H:%M:%S)\""
  }
}
```

(Use `-v+4M -v+40S` instead of `-d '...'` on stock macOS.)

Or project-specific in `.claude/settings.json`.

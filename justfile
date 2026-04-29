mod unifs "internal/cmd/unifs/justfile"
mod go "tool/go/justfile"

piper := 'TTY=0 piper -p piper.cue' + if env("DEBUG", "0") == '1' { ' --log-level=debug' } else { '' }

ship:
    {{ piper }} do ship push

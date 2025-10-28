#!/bin/bash
# [shoreman](https://github.com/chrismytton/shoreman) is an
# implementation of the **Procfile** format. Inspired by the original
# [foreman](http://ddollar.github.com/foreman/) tool for ruby.
# Copyright (c) 2011 Chris Mytton
# Copyright (c) 2025 Armin Ronacher (and Claude)
# 
# Permission is hereby granted, free of charge, to any person obtaining
# a copy of this software and associated documentation files (the
# "Software"), to deal in the Software without restriction, including
# without limitation the rights to use, copy, modify, merge, publish,
# distribute, sublicense, and/or sell copies of the Software, and to
# permit persons to whom the Software is furnished to do so, subject to
# the following conditions:
# 
# The above copyright notice and this permission notice shall be
# included in all copies or substantial portions of the Software.
# 
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
# EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
# MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
# NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
# LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
# OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
# WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

# Make sure that any errors cause the script to exit immediately.
set -eo pipefail
[[ "$TRACE" ]] && set -x

# ## Usage

# Usage message that is displayed when `--help` is given as an argument.
usage() {
  echo "Usage: shoreman [procfile|Procfile] [envfile|.env]"
  echo "Run Procfiles using shell."
  echo
  echo "The shoreman script reads commands from [procfile] and starts up the"
  echo "processes that it describes."
}

# ## Logging

# For logging we want to prefix each entry with the current time, as well
# as the process name. This takes two arguments, the name of the process
# with its index, and then reads data from stdin, formats it, and sends it
# to stdout and dev.log.
log() {
  local index="$2"
  local format="%s %s | %s"
  local log_format="%s %s | %s"
  
  # Maximum log line length (configurable via environment variable)
  local max_length="${MAX_LOG_LINE_LENGTH:-500}"

  # We add colors when output is a terminal. `SHOREMAN_COLORS` can override it.
  if [ -t 1 -o "$SHOREMAN_COLORS" == "always" ] \
     && [ "$SHOREMAN_COLORS" != "never" ]; then
    # Bash colors start from 31 up to 37. We calculate what color the process
    # gets based on its index.
    local color="$((31 + (index % 7)))"
    format="\033[0;${color}m%s %s |\033[0m %s"
  fi

  while IFS= read -r data
  do
    local timestamp="$(date +"%H:%M:%S")"
    
    # Truncate long log entries
    if [ ${#data} -gt $max_length ]; then
      data="${data:0:$max_length}..."
    fi
    
    printf "$format\n" "$timestamp" "$1" "$data"
    printf "$log_format\n" "$timestamp" "$1" "$data" >> dev.log
  done
}

# ## Running commands

# When a process is started, we want to keep track of its pid so we can
# `kill` it when the parent process receives a signal, and so we can `wait`
# for it to finish before exiting the parent process.
store_pid() {
  pids="$pids $1"
}

# This starts a command asynchronously and stores its pid in a list for use
# later on in the script.
start_command() {
  bash -c "$1" 2>&1 | log "$2" "$3" &
  pid="$(jobs -p %%)"
  store_pid "$pid"
}

# ## Reading the .env file

# The .env file needs to be a list of assignments like in a shell script.
# Shell-style comments are permitted.
load_env_file() {
  local env_file=${1:-'.env'}

  if [[ -f "$env_file" ]]; then
    export $(grep "^[^#]*=.*" "$env_file" | xargs)
  fi
}

# ## Reading the Procfile

# The Procfile needs to be parsed to extract the process names and commands.
# The file is given on stdin, see the `<` at the end of this while loop.
run_procfile() {
  local procfile=${1:-'Procfile'}
  # We give each process an index to track its color. We start with 1,
  # because it corresponds to green which is easier on the eye than red (0).
  local index=1
  while read line || [[ -n "$line" ]]; do
    if [[ -z "$line" ]] || [[ "$line" == \#* ]]; then continue; fi
    local name="${line%%:*}"
    local command="${line#*:[[:space:]]}"
    start_command "$command" "${name}" "$index"
    echo "'${command}' started with pid $pid" | log "${name}" "$index"
    index=$((index + 1))
  done < "$procfile"
}

# ## Cleanup

# When a `SIGINT`, `SIGTERM` or `EXIT` is received, this action is run, killing the
# child processes. The sleep stops STDOUT from pouring over the prompt, it
# should probably go at some point.
onexit() {
  echo "SIGINT received" > /dev/stderr
  echo "sending SIGTERM to all processes" > /dev/stderr
  echo "$(date +"%H:%M:%S") PROCESS MANAGER SHUTTING DOWN" >> dev.log
  kill $pids 2> /dev/null
  # Remove PID file on exit
  rm -f ".shoreman.pid"
  sleep 1
}

main() {
  local procfile="$1"
  local env_file="$2"
  local pidfile=".shoreman.pid"

  # If the `--help` option is given, show the usage message and exit.
  expr -- "$*" : ".*--help" >/dev/null && {
    usage
    exit 0
  }

  # Check if shoreman is already running
  if [[ -f "$pidfile" ]]; then
    local existing_pid=$(cat "$pidfile")
    if kill -0 "$existing_pid" 2>/dev/null; then
      echo "error: services are already running. that's good, we autoreload. no need to do anything" >&2
      exit 1
    fi
    # PID file exists but process is not running, remove stale PID file
    rm -f "$pidfile"
  fi

  # Write our PID to the PID file
  echo $$ > "$pidfile"

  # Move the current log to the previous run log
  if [[ -f dev.log ]]; then
    cp dev.log dev-prev.log
  fi
  > dev.log

  echo "!!! =================================================================" >> dev.log
  echo "$(date +"%H:%M:%S") SHOREMAN STARTED" >> dev.log
  echo "!!! =================================================================" >> dev.log

  load_env_file "$env_file"
  run_procfile "$procfile"

  trap onexit INT TERM

  exitcode=0
  for pid in $pids; do
      # Wait for the children to finish executing before exiting.
      wait "${pid}" 2> /dev/null || exitcode=$?
  done
  # Remove PID file on normal exit
  rm -f "$pidfile"
  exit $exitcode
}

main "$@"
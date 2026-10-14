#!/bin/bash
# Legacy Bash CLI retained for reference; the maintained CLI is in cmd/slingr.
# shellcheck disable=SC2016
set -e
# export PATH="/bin:/usr/bin:/usr/local/bin" # safer, if you can

(( BASH_VERSINFO[0] < 4 )) && echo "Bash 4+ required." && exit 1

: "${PAGER:=more}"
: "${EDITOR:=vi}"
: "${HELP_BROWSER:=}"
: "${EXE:="${0##*/}"}"

declare -A HELP
declare -A CONF

declare black="\e[30m"
declare red="\e[31m"
declare green="\e[32m"
declare yellow="\e[33m"
declare blue="\e[34m"
declare magenta="\e[35m"
declare cyan="\e[36m"
declare white="\e[37m"
declare reset="\e[0m"

_initialize() {
  APP="$(x.config app)"
  if [[ -z "$APP" ]]; then
    APP="$(_read_input 'App')"
    x.config 'app' "$APP"
  fi
  ENV="$(x.config env)"
  if [[ -z "$ENV" ]]; then
    ENV="$(_read_input 'Environment (dev|staging|prod)')"
    if [[ "$ENV" =~ ^(dev|prod|staging)$ ]]; then
      x.config 'env' "$ENV"
    else
      _initialize
    fi
  fi
  EMAIL="$(x.config email)"
  if [[ -z "$EMAIL" ]]; then
    EMAIL="$(_read_input 'Email')"
    x.config 'email' "$EMAIL"
  fi

  TOKEN="$(x.config "$APP.$ENV.token")"
  URL="https://$APP.slingrs.io/$ENV/runtime/api"
  MONITOR_URL="https://$APP.slingrs.io/$ENV/monitor/api"
}

_alternatives() {
  if [[ -z "$CMD" ]]; then
    x.current
  elif [[ "$CMD" == "add" ]]; then
    x.new "$@"
  elif [[ "$CMD" == "q" ]]; then
    x.query "$@"
  fi
}

HELP[main]='
# Slingr - bash command

CLI interface to interact with any slingr provider API.'

# ------- SHORCUTS

HELP[token]="Prints token $APP URL"
x.token() { echo "$TOKEN"; }

# ------- URL

HELP[runtime]="Prints runtime $APP URL"
x.runtime() { echo "📦 https://$APP.slingrs.io/$ENV/runtime"; }
HELP[monitor]="Prints monitor $APP URL"
x.monitor() { echo "💻 https://$APP.slingrs.io/$ENV/monitor"; }
HELP[builder]="Prints builder $APP URL"
x.builder() { echo "🔨 https://$APP.slingrs.io/$ENV/builder"; }

HELP[portal]='Prints Developer Portal URL'
x.portal(){ echo "🚪 https://developer-portal.slingrs.io/login.html"; }
HELP[docs]='Prints Platform Docs URL'
x.docs(){ echo "📘 https://platform-docs.slingr.io/"; }
HELP[web]='Prints Slingr Website'
x.web() { echo "🪃 https://www.slingr.io"; }

# ------- CONFIGURATIONS

HELP[email]="Configure user login email"

x.email() {
 if [[ -z "$1" ]]; then
   x.config email && exit
 fi
  x.config email "$1"
}

HELP[app]="Configure app name"

x.app() {
 if [[ -z "$1" ]]; then
   x.config app && exit
 fi
  x.config app "$1"
}

HELP[env]="Configure app environment"

x.env() {
  if [[ -z "$1" ]]; then
    x.config env && exit
  fi
  if [[ "$1" =~ ^(dev|prod|staging)$ ]]; then
    x.config 'env' "$1" && exit
  fi
  echo "Error: Invalid environment" && exit 1
}

HELP[current]='Prints the current app and environment'

x.current() { echo "$APP"-"$ENV"; }

# ------- AUTHENTICATION

HELP[login]="Login to $APP provision"

x.login() {
  password="$1"
  if [[ -z "$password" ]]; then
    password="$(_read_secret 'Password')"
  fi

  res="$(curl -s -X POST "$URL/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"'"$EMAIL"'","password":"'"$password"'"}')"
  echo "$res" | x.check.error > '/dev/null'

  TOKEN="$(jq -r '.token' <<< "$res")"
  PERSON="$(jq -r '.userName' <<< "$res")"
  x.config "$APP.$ENV.token" "$TOKEN"
  x.config "$APP.$ENV.person" "$PERSON"
  echo ""
}

x.impersonate() {
  password="$(_read_secret 'Password')"
  _check_param "impersonated-email" "$1"
  impersonate="$1"

  res="$(curl -s -X POST "$URL/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"'"$EMAIL"'","password":"'"$password"'","impersonated":"'"$impersonate"'"}')"

  TOKEN="$(jq -r '.token' <<< "$res")"
  PERSON="$(jq -r '.userName' <<< "$res")"
  x.config "$APP.$ENV.token" "$TOKEN"
  x.config "$APP.$ENV.person" "$PERSON"
}

HELP[logout]='Log out from slingr provision'

x.logout() {
  res="$(curl -s -X POST "$URL/auth/logout" \
    -H "Content-Type: application/json" \
    -H "token: $TOKEN")"
  echo "$res" | x.check.error
  x.config.delete "$APP.$ENV.token"
}

# ------- USER

HELP[user.current]='Returns current user information'

x.whoami() {
  curl -s "$URL/users/current" \
    -H "Content-Type: application/json" \
    -H "token: $TOKEN"
}

HELP[user.groups]='Returns the current user groups'

x.user.groups() {
  curl -s "$URL/users/currentGroups" \
    -H "Content-Type: application/json" \
    -H "token: $TOKEN"
}

# ------- STATUS

x.user.status() {
  curl -s "$URL/ui/default/metadata/provisions/statusForUser" \
    -H "Content-Type: application/json" \
    -H "token: $TOKEN"
}

# ------- PLATFORM MANAGER

HELP[clear.cache]='Clear the cache from the slingr app environment'

x.clear.cache() {
  res="$(curl -s -X PUT "$URL/system/clearCache" \
    -H "token: $TOKEN")"
  echo "$res"
}

HELP[fix.metadata]='Starts a job to fix the metadata'

x.fix.metadata() {
  res="$(curl -s -X PUT "$URL/metadata/fixAll" \
    -H "token: $TOKEN")"
  echo "$res"
}

# ------- DATA

HELP[get]='Get a record by entity name and id'

x.get() {
  _checkdep jq || return $?
  _check_param "entityName" "$1"
  _check_param "id" "$2"
  entity="$1"
  id="$2"
  res="$(curl -s -G "$URL/data/$entity/$id" \
    -H "Content-Type: application/json" \
    -H "token: $TOKEN"
  )"
  echo "$res"
}

HELP[query]='Get records by entity name and query'

x.query() {
  _checkdep jq || return $?
  _check_param "entityName" "$1"

  test -z "$entity" && entity="$1"; shift
  test -z "$query" && query="$(_flags_to_params "$@")"
  res="$(curl -s -G "$URL/data/$entity?$query" \
    -H "Content-Type: application/json" \
    -H "token: $TOKEN")"
  echo "$res"
}

HELP[aggregate]='Make an Aggregate Query'

x.aggregate() {
  _checkdep jq || return $?
  _check_param "entityName" "$1"

  test -z "$entity" && entity="$1"; shift
  test -z "$payload" && payload="$1"; shift
  res="$(curl -s -X PUT "$URL/data/$entity/aggregate" \
    -H "Content-Type: application/json" \
    -H "token: $TOKEN" \
    -d "$payload")"
  echo "$res"
}

HELP[new]='Get a record by entity name and id'

x.new() {
  _checkdep jq || return $?
  _check_param "entityName" "$1"
  entity="$1"
  payload="$2"
  if [[ -z "$payload" ]]; then
    payload='{}'
  fi
  res="$(curl -s -X POST "$URL/data/$entity" \
    -H "Content-Type: application/json" \
    -H "token: $TOKEN" \
    -d "$payload"
  )"
  echo "$res"
}

HELP[rm]='Remove a record by entity name and id'

x.rm() {
  _checkdep jq || return $?
  _check_param "entityName" "$1"
  _check_param "id" "$2"
  entity="$1"
  id="$2"
  res="$(curl -s -X DELETE "$URL/data/$entity/$id" \
    -H "Content-Type: application/json" \
    -H "token: $TOKEN"
  )"
  echo "$res"
}

# TO DO
# x.rm.query entityName --params
# x.edit entityName id
# x.do entityName actionName id payload?
# x.do.query entityName actionName action --params?
# x.do.global entityName actionName payload?
# x.download fileId

x.download() {
  res="$(curl -s -G "$URL/files/$1" \
    -H "token: $TOKEN"
  )"
  echo "$res"
}

# x.upload filePath

## Logging

x.logs.get() {
  res="$(curl -s -G "$URL/status/logs" \
    -H "Content-Type: application/json" \
    -H "token: $TOKEN" \
    --data-urlencode "_offset=$1"
  )"
  echo "$res"
}

x.logs.send() {
  curl -s -X POST "$URL/status/logs" \
    -H "Content-Type: application/json" \
    -H "token: $TOKEN" \
    --data '{"level":"'"$1"'","message":"'"$2"'"}'
}

x.logs.info() { x.logs.send 'INFO' "$*"; }
x.logs.warn() { x.logs.send 'WARN' "$*"; }
x.logs.error() { x.logs.send 'ERROR' "$*"; }
x.logs.debug() { x.logs.send 'DEBUG' "$*"; }

x.logs.watch() {
  while :; do
    res="$(x.logs.get "$offset")"
    offset="$(jq -r '.items[-1].id' <<< "$res")"
    jq -r '.items[] | if .level == "INFO" then "\u001b[0m" else "\u001b[31m" end +
    "["+.level+"]["+.userEmail+"] "+.message' <<< "$res"
    read -t 0.01 -r -s -N 1 && [[ $REPLY == 'q' ]] && break
  done
}

# Executor

x.console() {
  file="$1"
  if [[ -z "$1" ]]; then
    file="$APP-$ENV-script"
  fi
  "$EDITOR" "/tmp/$file.js"
  printf "[$APP-$ENV] Execute script? (y/N) "
  read -r r
  if [[ $r == "y" ]]; then
    x.console.execute
    exit
  fi
}

consoleLibrary='
const log = (input) => {
  let out = ""; 
  if (typeof input === "object") {
    out = JSON.stringify(input); 
  } else if (typeof input === "string") {
    out = input;
  } else if (typeof msg !== "undefined") {
    out = JSON.stringify(input);
  }
  sys.jobs.logInfo(out);
};

const logLine = () => {
  sys.jobs.logInfo("------------------");
}
'
x.console.execute() {
  s="$(jq -n \
    --arg script "$consoleLibrary$(cat "/tmp/$file.js")" \
    '.script=$script')"
  res="$(curl -s -X POST "$MONITOR_URL/scripts/executor" \
    -H 'Content-Type: application/json' \
    -H "token: $TOKEN" \
    --data "$s")"
    x.console.watch "$(jq -r '.jobId' <<< "$res")"
}

x.console.watch() {
  job="$1"
  offset=0
  while
    res="$(curl -s "$MONITOR_URL/status/jobs/$job/logs?_size=200&_offset=$offset" -H 'Content-Type: application/json' -H "token: $TOKEN")"
    offset="$(jq -r '.offset' <<< "$res")"
    #jq -r \
    #  '.items[] | if .level == "ERROR" then "\u001b[31m" else "\u001b[0m" end +
    #      "["+ (.timestamp / 1000 | strflocaltime("%H:%M:%S")) +"] " +
    #      if (.message | startswith("{")) then (.message | fromjson) else .message end + "\u001b[0m"' <<< "$res"

    echo "$res" | jq -r '.items[] | [.timestamp, .message, .level] | @tsv' | while IFS=$'\t' read -r timestamp message level; do
      seconds="$((timestamp / 1000))"
      #nanoseconds="$(((timestamp % 1000) * 1000000))"
      nanoseconds="$((timestamp % 1000))"
      t="$(printf "%02d:%02d:%02d.%.3d" $(date -d @$seconds +"%H %M %S") $nanoseconds)"
      if [[ "$message" == "Running script" ]]; then
        printf "\u001b[33m[$t] $message\u001b[0m\n"
      elif [[ "$message" =~ "Script run"  ]]; then
        printf "\u001b[32m[$t] $message\u001b[0m\n"
      elif [[ "$message" =~ "Script contains"  ]]; then
        printf "\u001b[33m[$t] $message\u001b[0m\n"
      elif [[ "$message" =~ ^\{.*\} ]]; then
        json="$(jq -C '.' <<< $message)"
        printf "[$t] $json\n"
      elif [[ "$level" == "ERROR" ]]; then
        printf "\u001b[31m[$t] $message\u001b[0m\n"
      else
        msg="$(awk '
          BEGIN {
            FS="[][]"
          }
          {
            for (i=1; i<=NF; i++) {
              if (i % 2 == 0) {
                printf "[\033[32m%s\033[0m]", $i
              } else {
                printf "%s", $i
              }
            }
            print ""
          }
          ' <<< $message)"
        printf "\u001b[0m[$t] $msg\u001b[0m\n"
      fi
    done

    res="$(curl -s "$MONITOR_URL/status/jobs/$job" -H 'Content-Type: application/json' -H "token: $TOKEN")"
    job_status="$(jq -r '.status' <<< "$res")"

    [[ "$offset" != "0" && "$job_status" != "FINISHED" ]]
  do true; done
}

_send_custom_error() {
  jq -n --arg code "$1" --arg message "$2" '.code=$code|.message=$message' && exit 1
}

x.check.error() {
  read response
  has_message="$(jq 'has("message")' <<< "$response")"
  if [[ "$has_message" == "true" ]]; then
    echo "[$APP-$ENV] Error: $(jq -r '.message' <<< "$response")" >&2
    exit 1
  fi
  echo "$response"
}

_check_token() {
  has_message="$(jq 'has("message")' <<< "$1")"
  test "$has_message" == "false" && return
  if [[ "$(jq '.message | startswith("Invalid token.")' <<< "$1")" == "true" ]]; then
     x.login && return
  fi
}

_check_param() {
  if [[ -z "$2" ]]; then
    echo "Error: Parameter [$1] is require" && exit 1
    #_send_custom_error "invalidParam" "$EXE: Parameter <$1> is needed" && exit 1
  fi
}

_flags_to_params() {
  while test $# -gt 0; do
    case "$1" in
      *)  
        key="$(echo $1 | sed -e 's/^\-\-//g')"
        test ! -z "$2" && value="$(_urlencode "$2")"
        query="$query$key=$value&"
        shift && shift
        ;;
    esac
  done
  echo "$query" | sed -e 's/\&$//'
}

_read_input() {
  local input
  while true; do
    read -p "$1: " input
    if [[ -z "$input" ]]; then
      echo "$1 cannot be empty. Please try again."
    else
      break
    fi
  done
  echo "$input"
}

_read_secret() {
  local input
  while true; do
    read -sp "$1: " input
    if [[ -z "$input" ]]; then
      echo "$1 cannot be empty. Please try again."
    else
      break
    fi
  done
  echo "$input"
}

############################## BOILERPLATE ###########################
## Everything from here to the end of file can be snipped and updated
## with latest from https://github.com/rwxrob/template-bash-command.

# ------------------------------- usage ------------------------------

HELP[usage]='
Display all possible commands. Note that this is usually easier by
simply using tab completion instead.'

x.usage() {
  local -a cmds
  for c in "${COMMANDS[@]}"; do
    [[ ${c:0:1} =~ _ ]] && continue
    cmds+=("$c")
  done
  local IFS='|'
  printf "usage: %s (%s)\n" "$EXE" "${cmds[*]}"
}

# ------------------------------- help -------------------------------

HELP[help]='

```
'"$EXE"' help [COMMAND]
```

Displays specific help information. If no argument is passed displays
general help information (main). Otherwise, the documentation for the
specific argument keyword is displayed, which usually corresponds to
a COMMAND name (but not necessarily). All documentation is written in
GitHub Flavored Markdown and will displayed as a web page if `pandoc`
and `$HELP_BROWSER` are detected, otherwise, just the Markdown is sent
to `$PAGER` (default: more).

Also see `readme` and `usage` commands.
'

x.help() { 
  local name="${1:-main}" title body file
  title=$(_help_title "$name") || true
  if [[ -z "$title" ]]; then
    title="$EXE $name"
    [[ $name == main ]] && title="$EXE"
  fi
  if [[ $name == main ]]; then
    body=$(x.readme)
    body=${body#*$title}
  else
    body="${HELP[$name]}"
  fi
  file="/tmp/help-$EXE-$name.html"
  if _have pandoc ; then
    if _have "$HELP_BROWSER" && [[ -t 1 ]] ;then
      pandoc -f gfm -s --metadata title="$title" \
        -o "$file" <<< "$body"
      [[ -z "$2" ]] && cd /tmp && exec "$HELP_BROWSER" "$file"
      return 0
    fi
    pandoc -f gfm -s --metadata title="$title" \
      -t plain <<< "$body" | "$PAGER"
    return 0
  fi
  echo -e "$title\n\n$body" | "$PAGER"
}

_help_title() {
  _filter "$@" && return $?;
  local name="$1"
  while IFS= read -r line; do
    [[ $line =~ ^[:space]*$ ]] && continue
    [[ $line =~ ^#\ (.+) ]] && echo "${BASH_REMATCH[1]}" && return 0
    return 1
  done <<< "${HELP[$name]}"
}

# ------------------------------ readme ------------------------------

HELP[readme]='
## Generate `README.md` File

```
'"$EXE"' readme > README.md
```

The `readme` command will output the embedded help documentation in raw
GitHub Flavored Markdown suitable for use as a `README.md` file on
GitHub or similar hosting service.'

x.readme() {
  _trim "${HELP[main]}"
  echo
  while IFS= read -r name; do
    [[ $name = main ]] && continue
    body=$(_trim "${HELP[$name]}")
    [[ $body =~ ^\# ]] || body="## The \`$name\` Command"$'\n\n'$body
    printf "%s\n\n" "$body"
  done < <(printf "%s\n" "${!HELP[@]}" | LC_COLLATE=C sort)
  echo -e "----\n\n*Autogenerated $(date)*\n"
}

# ------------------------------ config ------------------------------

HELP[config]='

```
'"$EXE"' config
'"$EXE"' config KEY
'"$EXE"' config.set KEY VALUE
'"$EXE"' config.set KEY ""
'"$EXE"' config.keys
'"$EXE"' config.values
'"$EXE"' config.directory
'"$EXE"' config.path [file]
'"$EXE"' config.edit [file]
'"$EXE"' config.delete
'"$EXE"' config.read
'"$EXE"' config.write
'"$EXE"' config.dump
```

The `config` command is for reading, writing, and displaying standard
open desktop configuration properties. 

### Arguments

With no arguments calls `dump` and outputs all the currently cached
configuration settings.

With a single KEY argument fetches the value for that key and outputs
it unless it is one of the following special (reserved) key names:

* `directory` full path to config directory
* `path`   full path to specific config file (default: `values`) 
* `edit`   opens config file in editor (default: `editor` or `$EDITOR)
* `keys`   output the configuration keys, one per line
* `values` output the configuration values, one per line
* `delete` if key argument then delete a specific key, otherwise prompt
* `read` reads the configuration file into CONF associative array
* `write` write the CONF associative array to the configuration file
* `dump` write the flattened CONF associative array to standard output

With more than one argument the remaining arguments after the KEY will
be combined into the VALUE and written to a `values` file in the
configuration directory. 

### Configuration Directory

The configuration directory path relies on the following environment
variables:

* `EXE` - defaults to name of currently running command ('"$EXE"')
* `HOME` - checked for `$HOME/.config/$EXE/values`
* `XDG_CONFIG_HOME` - overrides `$HOME/.config`
* `CONFIG_DIR` - full path to directory containing `values` file

The `CONFIG_DIR` always takes priority over anything else if set, but is
never implied. If the directory does not exist it will be created the
first time a value is set.

### Configuration `values` File Format

The file (which is almost always located at
`~/.config/'"$EXE"'/values`) uses the simplest possible format to
facilitate standard UNIX parsing and filtering with any number of
existing tools (and no `jq` dependency).

* One KEY=VALUE per line
* KEYs may be anything but the equal sign (`=`)
* VALUEs may be anything but line returns must be escaped

Note that, although similar, this is *not* the same as Java properties
and other similar format. It is designed for ultimate simplicity,
efficiency, and portability.'

x.config() {
  case $# in
    0) x.config.dump ;;
    1) x.config.get "$@" ;;
    *) x.config.set "$@" ;;
  esac
}

x.config.edit() {
  : "${CONF[editor]:="${EDITOR:=vi}"}"
  exec "${CONF[editor]}" "$(x.config.path "${1:-values}")"
}

x.config.delete() {
  if [[ -z "$1" ]];then
    select key in "${!CONF[@]}"; do
      x.config.delete "$key"
      return $? 
    done
  fi
  x.config.set "$1" ''
}

x.config.keys() { printf "%s\n" "${!CONF[@]}"; }

x.config.values() { printf "%s\n" "${CONF[@]}"; }

x.config.dir() {
  local dir="$HOME/.config/$EXE"
  [[ -n "$XDG_CONFIG_HOME" ]] && dir="$XDG_CONFIG_HOME/$EXE" 
  [[ -n "$CONFIG_DIR" ]] && dir="$CONFIG_DIR"
  [[ -n "$1" ]] && echo "$dir/$1" && return 0
  printf "%s" "$dir"
  [[ -t 1 ]] && echo
  return 0
}

x.config.path() { 
  local file=${1:-values}
  printf "%s/%s" "$(x.config.dir)" "$file"
  [[ -t 1 ]] && echo
  return 0
}

x.config.set() {
  local key="$1"; shift; local val="$*"
  val="${val//$'\n'/\\n}"
  CONF["$key"]="$val"
  x.config.write
}

x.config.get() { 
  printf "%s" "${CONF[$1]}"
  [[ -t 1 ]] && echo
  return 0
}

x.config.read() {
  local values="$(x.config.path)"
  [[ -r "$values" ]] || return 0
  while IFS= read -r line; do
    [[ $line =~ ^([^=]+)=(.+)$ ]] || continue
    CONF["${BASH_REMATCH[1]}"]="${BASH_REMATCH[2]}"
  done < "$values"
}

x.config.write() {
  local dir="$(x.config.dir)"
  mkdir -p "$dir"
  x.config.dump > "$dir/values"
}

x.config.dump() {
  (( ${#CONF[@]} == 0 )) && return 0
  paste -d=\
    <(printf "%s\n" "${!CONF[@]}") \
    <(printf "%s\n" "${CONF[@]}") \
  | sort
}

# ----------------------------- utilities ----------------------------

_jsonstr() {
  _checkdep jq || return $?
  _buffer "$@" && return $?
  jq -MRsc <<< "$1"
}

_urlencode() {
  _buffer "$@" && return $?
  local string="$1"
  local strlen=${#string}
  local encoded=""
  local pos c o
  for ((pos = 0; pos < strlen; pos++)); do
    c=${string:$pos:1}
    case "$c" in
    [-_.~a-zA-Z0-9]) o="$c" ;;
    *) printf -v o '%%%02x' "'$c'" ;;
    esac
    encoded+="$o"
  done
  echo "$encoded"
}

_reduce() {
  local -n name="${1:?"name of array required"}"
  while IFS= read -r key; do
    [[ $key =~ $2 ]] && echo "$key"
  done < <(printf "%s\n" "${name[@]}")
}

_newest() {
  IFS=$'\n'
  mapfile -t f < <(ls -1 --color=never -trd "${1:-.}"/* 2>/dev/null)
  [[ ${#f} > 0 ]] && echo "${f[-1]}"
}

_trim() {
  local it="${1#"${1%%[![:space:]]*}"}"
  echo -e "${it%"${it##*[![:space:]]}"}"
}

_join() { local IFS="$1"; shift; echo "$*"; }

_have(){ type "$1" &>/dev/null; }

_checkdep() {
  _have "$1" && return 0
  echo "'$EXE' depends on '$1' for this, but not found"
  return 1
}

_filter(){
  [[ -n "$1" ]] && return 1
  while IFS= read -ra args; do
    "${FUNCNAME[1]}" "${args[@]}"
  done
}

_buffer() {
  [[ -n "$1" ]] && return 1
  "${FUNCNAME[1]}" "$(</dev/stdin)"
}

_prompt() {
  local key="$1" def="$2" regx="$3" value first=yes
  shift 3
  local text="${*:-Enter value for %s [%s]: }"
  [[ -z "$key" ]] && echo "Missing prompt key" >&2 && return 1
  [[ -z "$regx" ]] && echo "Missing valid regx" >&2 && return 1
  while [[ ! $value =~ $regx ]];do
    printf "$text" "$key" "$def" >&2
    IFS= read -r value
    [[ -z "$value" ]] && value="$def"
    [[ $value =~ ^\ +$ ]] && value=""
    [[ -n "$first" ]] && unset first && continue
    echo "Must match /$regx/" >&2
  done
  _trim "$value"
}

# --------------------- completion and delegation --------------------
#      `complete -C foo foo` > `source <(foo bloated_completion)`

x.config.read
_have _initialize && _initialize "$@"

while IFS= read -r line; do
  [[ $line =~ ^declare\ -f\ x\. ]] || continue
  COMMANDS+=( "${line##declare -f x.}" )
done < <(declare -F)
mapfile -t COMMANDS < \
  <(LC_COLLATE=C sort < <(printf "%s\n" "${COMMANDS[@]}"))

if [[ -n $COMP_LINE ]]; then
  line=${COMP_LINE#* }
  for c in "${COMMANDS[@]}"; do
    [[ ${c:0:${#line}} == "${line,,}" ]] && echo "$c"
  done
  exit
fi

for c in "${COMMANDS[@]}"; do
  if [[ $c == "$EXE" ]]; then
    "x.$EXE" "$@"
    exit $?
  fi
done

if [[ -n "$1" ]]; then
  declare CMD="$1"; shift
  for c in "${COMMANDS[@]}"; do
    declare cmd=$(command -v "x.$c")
    if [[ $c == "$CMD" && -n "$cmd" ]]; then
      "x.$CMD" "$@"
      exit $?
    fi
  done
fi

if _have _alternatives; then
  _alternatives "$@" 
  exit $?
fi

x.usage "$@"

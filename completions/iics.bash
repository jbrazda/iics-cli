# bash completion for iics                                 -*- shell-script -*-

__iics_debug()
{
    if [[ -n ${BASH_COMP_DEBUG_FILE:-} ]]; then
        echo "$*" >> "${BASH_COMP_DEBUG_FILE}"
    fi
}

# Homebrew on Macs have version 1.3 of bash-completion which doesn't include
# _init_completion. This is a very minimal version of that function.
__iics_init_completion()
{
    COMPREPLY=()
    _get_comp_words_by_ref "$@" cur prev words cword
}

__iics_index_of_word()
{
    local w word=$1
    shift
    index=0
    for w in "$@"; do
        [[ $w = "$word" ]] && return
        index=$((index+1))
    done
    index=-1
}

__iics_contains_word()
{
    local w word=$1; shift
    for w in "$@"; do
        [[ $w = "$word" ]] && return
    done
    return 1
}

__iics_handle_go_custom_completion()
{
    __iics_debug "${FUNCNAME[0]}: cur is ${cur}, words[*] is ${words[*]}, #words[@] is ${#words[@]}"

    local shellCompDirectiveError=1
    local shellCompDirectiveNoSpace=2
    local shellCompDirectiveNoFileComp=4
    local shellCompDirectiveFilterFileExt=8
    local shellCompDirectiveFilterDirs=16

    local out requestComp lastParam lastChar comp directive args

    # Prepare the command to request completions for the program.
    # Calling ${words[0]} instead of directly iics allows handling aliases
    args=("${words[@]:1}")
    # Disable ActiveHelp which is not supported for bash completion v1
    requestComp="IICS_ACTIVE_HELP=0 ${words[0]} __completeNoDesc ${args[*]}"

    lastParam=${words[$((${#words[@]}-1))]}
    lastChar=${lastParam:$((${#lastParam}-1)):1}
    __iics_debug "${FUNCNAME[0]}: lastParam ${lastParam}, lastChar ${lastChar}"

    if [ -z "${cur}" ] && [ "${lastChar}" != "=" ]; then
        # If the last parameter is complete (there is a space following it)
        # We add an extra empty parameter so we can indicate this to the go method.
        __iics_debug "${FUNCNAME[0]}: Adding extra empty parameter"
        requestComp="${requestComp} \"\""
    fi

    __iics_debug "${FUNCNAME[0]}: calling ${requestComp}"
    # Use eval to handle any environment variables and such
    out=$(eval "${requestComp}" 2>/dev/null)

    # Extract the directive integer at the very end of the output following a colon (:)
    directive=${out##*:}
    # Remove the directive
    out=${out%:*}
    if [ "${directive}" = "${out}" ]; then
        # There is not directive specified
        directive=0
    fi
    __iics_debug "${FUNCNAME[0]}: the completion directive is: ${directive}"
    __iics_debug "${FUNCNAME[0]}: the completions are: ${out}"

    if [ $((directive & shellCompDirectiveError)) -ne 0 ]; then
        # Error code.  No completion.
        __iics_debug "${FUNCNAME[0]}: received error from custom completion go code"
        return
    else
        if [ $((directive & shellCompDirectiveNoSpace)) -ne 0 ]; then
            if [[ $(type -t compopt) = "builtin" ]]; then
                __iics_debug "${FUNCNAME[0]}: activating no space"
                compopt -o nospace
            fi
        fi
        if [ $((directive & shellCompDirectiveNoFileComp)) -ne 0 ]; then
            if [[ $(type -t compopt) = "builtin" ]]; then
                __iics_debug "${FUNCNAME[0]}: activating no file completion"
                compopt +o default
            fi
        fi
    fi

    if [ $((directive & shellCompDirectiveFilterFileExt)) -ne 0 ]; then
        # File extension filtering
        local fullFilter filter filteringCmd
        # Do not use quotes around the $out variable or else newline
        # characters will be kept.
        for filter in ${out}; do
            fullFilter+="$filter|"
        done

        filteringCmd="_filedir $fullFilter"
        __iics_debug "File filtering command: $filteringCmd"
        $filteringCmd
    elif [ $((directive & shellCompDirectiveFilterDirs)) -ne 0 ]; then
        # File completion for directories only
        local subdir
        # Use printf to strip any trailing newline
        subdir=$(printf "%s" "${out}")
        if [ -n "$subdir" ]; then
            __iics_debug "Listing directories in $subdir"
            __iics_handle_subdirs_in_dir_flag "$subdir"
        else
            __iics_debug "Listing directories in ."
            _filedir -d
        fi
    else
        while IFS='' read -r comp; do
            COMPREPLY+=("$comp")
        done < <(compgen -W "${out}" -- "$cur")
    fi
}

__iics_handle_reply()
{
    __iics_debug "${FUNCNAME[0]}"
    local comp
    case $cur in
        -*)
            if [[ $(type -t compopt) = "builtin" ]]; then
                compopt -o nospace
            fi
            local allflags
            if [ ${#must_have_one_flag[@]} -ne 0 ]; then
                allflags=("${must_have_one_flag[@]}")
            else
                allflags=("${flags[*]} ${two_word_flags[*]}")
            fi
            while IFS='' read -r comp; do
                COMPREPLY+=("$comp")
            done < <(compgen -W "${allflags[*]}" -- "$cur")
            if [[ $(type -t compopt) = "builtin" ]]; then
                [[ "${COMPREPLY[0]}" == *= ]] || compopt +o nospace
            fi

            # complete after --flag=abc
            if [[ $cur == *=* ]]; then
                if [[ $(type -t compopt) = "builtin" ]]; then
                    compopt +o nospace
                fi

                local index flag
                flag="${cur%=*}"
                __iics_index_of_word "${flag}" "${flags_with_completion[@]}"
                COMPREPLY=()
                if [[ ${index} -ge 0 ]]; then
                    PREFIX=""
                    cur="${cur#*=}"
                    ${flags_completion[${index}]}
                    if [ -n "${ZSH_VERSION:-}" ]; then
                        # zsh completion needs --flag= prefix
                        eval "COMPREPLY=( \"\${COMPREPLY[@]/#/${flag}=}\" )"
                    fi
                fi
            fi

            if [[ -z "${flag_parsing_disabled}" ]]; then
                # If flag parsing is enabled, we have completed the flags and can return.
                # If flag parsing is disabled, we may not know all (or any) of the flags, so we fallthrough
                # to possibly call handle_go_custom_completion.
                return 0;
            fi
            ;;
    esac

    # check if we are handling a flag with special work handling
    local index
    __iics_index_of_word "${prev}" "${flags_with_completion[@]}"
    if [[ ${index} -ge 0 ]]; then
        ${flags_completion[${index}]}
        return
    fi

    # we are parsing a flag and don't have a special handler, no completion
    if [[ ${cur} != "${words[cword]}" ]]; then
        return
    fi

    local completions
    completions=("${commands[@]}")
    if [[ ${#must_have_one_noun[@]} -ne 0 ]]; then
        completions+=("${must_have_one_noun[@]}")
    elif [[ -n "${has_completion_function}" ]]; then
        # if a go completion function is provided, defer to that function
        __iics_handle_go_custom_completion
    fi
    if [[ ${#must_have_one_flag[@]} -ne 0 ]]; then
        completions+=("${must_have_one_flag[@]}")
    fi
    while IFS='' read -r comp; do
        COMPREPLY+=("$comp")
    done < <(compgen -W "${completions[*]}" -- "$cur")

    if [[ ${#COMPREPLY[@]} -eq 0 && ${#noun_aliases[@]} -gt 0 && ${#must_have_one_noun[@]} -ne 0 ]]; then
        while IFS='' read -r comp; do
            COMPREPLY+=("$comp")
        done < <(compgen -W "${noun_aliases[*]}" -- "$cur")
    fi

    if [[ ${#COMPREPLY[@]} -eq 0 ]]; then
        if declare -F __iics_custom_func >/dev/null; then
            # try command name qualified custom func
            __iics_custom_func
        else
            # otherwise fall back to unqualified for compatibility
            declare -F __custom_func >/dev/null && __custom_func
        fi
    fi

    # available in bash-completion >= 2, not always present on macOS
    if declare -F __ltrim_colon_completions >/dev/null; then
        __ltrim_colon_completions "$cur"
    fi

    # If there is only 1 completion and it is a flag with an = it will be completed
    # but we don't want a space after the =
    if [[ "${#COMPREPLY[@]}" -eq "1" ]] && [[ $(type -t compopt) = "builtin" ]] && [[ "${COMPREPLY[0]}" == --*= ]]; then
       compopt -o nospace
    fi
}

# The arguments should be in the form "ext1|ext2|extn"
__iics_handle_filename_extension_flag()
{
    local ext="$1"
    _filedir "@(${ext})"
}

__iics_handle_subdirs_in_dir_flag()
{
    local dir="$1"
    pushd "${dir}" >/dev/null 2>&1 && _filedir -d && popd >/dev/null 2>&1 || return
}

__iics_handle_flag()
{
    __iics_debug "${FUNCNAME[0]}: c is $c words[c] is ${words[c]}"

    # if a command required a flag, and we found it, unset must_have_one_flag()
    local flagname=${words[c]}
    local flagvalue=""
    # if the word contained an =
    if [[ ${words[c]} == *"="* ]]; then
        flagvalue=${flagname#*=} # take in as flagvalue after the =
        flagname=${flagname%=*} # strip everything after the =
        flagname="${flagname}=" # but put the = back
    fi
    __iics_debug "${FUNCNAME[0]}: looking for ${flagname}"
    if __iics_contains_word "${flagname}" "${must_have_one_flag[@]}"; then
        must_have_one_flag=()
    fi

    # if you set a flag which only applies to this command, don't show subcommands
    if __iics_contains_word "${flagname}" "${local_nonpersistent_flags[@]}"; then
      commands=()
    fi

    # keep flag value with flagname as flaghash
    # flaghash variable is an associative array which is only supported in bash > 3.
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        if [ -n "${flagvalue}" ] ; then
            flaghash[${flagname}]=${flagvalue}
        elif [ -n "${words[ $((c+1)) ]}" ] ; then
            flaghash[${flagname}]=${words[ $((c+1)) ]}
        else
            flaghash[${flagname}]="true" # pad "true" for bool flag
        fi
    fi

    # skip the argument to a two word flag
    if [[ ${words[c]} != *"="* ]] && __iics_contains_word "${words[c]}" "${two_word_flags[@]}"; then
        __iics_debug "${FUNCNAME[0]}: found a flag ${words[c]}, skip the next argument"
        c=$((c+1))
        # if we are looking for a flags value, don't show commands
        if [[ $c -eq $cword ]]; then
            commands=()
        fi
    fi

    c=$((c+1))

}

__iics_handle_noun()
{
    __iics_debug "${FUNCNAME[0]}: c is $c words[c] is ${words[c]}"

    if __iics_contains_word "${words[c]}" "${must_have_one_noun[@]}"; then
        must_have_one_noun=()
    elif __iics_contains_word "${words[c]}" "${noun_aliases[@]}"; then
        must_have_one_noun=()
    fi

    nouns+=("${words[c]}")
    c=$((c+1))
}

__iics_handle_command()
{
    __iics_debug "${FUNCNAME[0]}: c is $c words[c] is ${words[c]}"

    local next_command
    if [[ -n ${last_command} ]]; then
        next_command="_${last_command}_${words[c]//:/__}"
    else
        if [[ $c -eq 0 ]]; then
            next_command="_iics_root_command"
        else
            next_command="_${words[c]//:/__}"
        fi
    fi
    c=$((c+1))
    __iics_debug "${FUNCNAME[0]}: looking for ${next_command}"
    declare -F "$next_command" >/dev/null && $next_command
}

__iics_handle_word()
{
    if [[ $c -ge $cword ]]; then
        __iics_handle_reply
        return
    fi
    __iics_debug "${FUNCNAME[0]}: c is $c words[c] is ${words[c]}"
    if [[ "${words[c]}" == -* ]]; then
        __iics_handle_flag
    elif __iics_contains_word "${words[c]}" "${commands[@]}"; then
        __iics_handle_command
    elif [[ $c -eq 0 ]]; then
        __iics_handle_command
    elif __iics_contains_word "${words[c]}" "${command_aliases[@]}"; then
        # aliashash variable is an associative array which is only supported in bash > 3.
        if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
            words[c]=${aliashash[${words[c]}]}
            __iics_handle_command
        else
            __iics_handle_noun
        fi
    else
        __iics_handle_noun
    fi
    __iics_handle_word
}

_iics_activitylog_get()
{
    last_command="iics_activitylog_get"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--fields=")
    two_word_flags+=("--fields")
    local_nonpersistent_flags+=("--fields")
    local_nonpersistent_flags+=("--fields=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_activitylog_list()
{
    last_command="iics_activitylog_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--fields=")
    two_word_flags+=("--fields")
    local_nonpersistent_flags+=("--fields")
    local_nonpersistent_flags+=("--fields=")
    flags+=("--limit=")
    two_word_flags+=("--limit")
    local_nonpersistent_flags+=("--limit")
    local_nonpersistent_flags+=("--limit=")
    flags+=("--offset=")
    two_word_flags+=("--offset")
    local_nonpersistent_flags+=("--offset")
    local_nonpersistent_flags+=("--offset=")
    flags+=("--run-id=")
    two_word_flags+=("--run-id")
    local_nonpersistent_flags+=("--run-id")
    local_nonpersistent_flags+=("--run-id=")
    flags+=("--task-id=")
    two_word_flags+=("--task-id")
    local_nonpersistent_flags+=("--task-id")
    local_nonpersistent_flags+=("--task-id=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_activitylog()
{
    last_command="iics_activitylog"

    command_aliases=()

    commands=()
    commands+=("get")
    commands+=("list")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_agent_details()
{
    last_command="iics_agent_details"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--id=")
    two_word_flags+=("--id")
    local_nonpersistent_flags+=("--id")
    local_nonpersistent_flags+=("--id=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_agent_get()
{
    last_command="iics_agent_get"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--id=")
    two_word_flags+=("--id")
    local_nonpersistent_flags+=("--id")
    local_nonpersistent_flags+=("--id=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_agent_list()
{
    last_command="iics_agent_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--limit=")
    two_word_flags+=("--limit")
    local_nonpersistent_flags+=("--limit")
    local_nonpersistent_flags+=("--limit=")
    flags+=("--skip=")
    two_word_flags+=("--skip")
    local_nonpersistent_flags+=("--skip")
    local_nonpersistent_flags+=("--skip=")
    flags+=("--unassigned")
    local_nonpersistent_flags+=("--unassigned")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_agent_start()
{
    last_command="iics_agent_start"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--id=")
    two_word_flags+=("--id")
    local_nonpersistent_flags+=("--id")
    local_nonpersistent_flags+=("--id=")
    flags+=("--service=")
    two_word_flags+=("--service")
    local_nonpersistent_flags+=("--service")
    local_nonpersistent_flags+=("--service=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_agent_stop()
{
    last_command="iics_agent_stop"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--id=")
    two_word_flags+=("--id")
    local_nonpersistent_flags+=("--id")
    local_nonpersistent_flags+=("--id=")
    flags+=("--service=")
    two_word_flags+=("--service")
    local_nonpersistent_flags+=("--service")
    local_nonpersistent_flags+=("--service=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_agent()
{
    last_command="iics_agent"

    command_aliases=()

    commands=()
    commands+=("details")
    commands+=("get")
    commands+=("list")
    commands+=("start")
    commands+=("stop")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_auditlog_list()
{
    last_command="iics_auditlog_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--fields=")
    two_word_flags+=("--fields")
    local_nonpersistent_flags+=("--fields")
    local_nonpersistent_flags+=("--fields=")
    flags+=("--limit=")
    two_word_flags+=("--limit")
    local_nonpersistent_flags+=("--limit")
    local_nonpersistent_flags+=("--limit=")
    flags+=("--skip=")
    two_word_flags+=("--skip")
    local_nonpersistent_flags+=("--skip")
    local_nonpersistent_flags+=("--skip=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_auditlog()
{
    last_command="iics_auditlog"

    command_aliases=()

    commands=()
    commands+=("list")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_completion_bash()
{
    last_command="iics_completion_bash"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_completion_fish()
{
    last_command="iics_completion_fish"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_completion_powershell()
{
    last_command="iics_completion_powershell"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_completion_zsh()
{
    last_command="iics_completion_zsh"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_completion()
{
    last_command="iics_completion"

    command_aliases=()

    commands=()
    commands+=("bash")
    commands+=("fish")
    commands+=("powershell")
    commands+=("zsh")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_connection_create()
{
    last_command="iics_connection_create"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--from-file=")
    two_word_flags+=("--from-file")
    local_nonpersistent_flags+=("--from-file")
    local_nonpersistent_flags+=("--from-file=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_connection_delete()
{
    last_command="iics_connection_delete"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--id=")
    two_word_flags+=("--id")
    local_nonpersistent_flags+=("--id")
    local_nonpersistent_flags+=("--id=")
    flags+=("--yes")
    flags+=("-y")
    local_nonpersistent_flags+=("--yes")
    local_nonpersistent_flags+=("-y")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_connection_get()
{
    last_command="iics_connection_get"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--id=")
    two_word_flags+=("--id")
    local_nonpersistent_flags+=("--id")
    local_nonpersistent_flags+=("--id=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_connection_list()
{
    last_command="iics_connection_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--limit=")
    two_word_flags+=("--limit")
    local_nonpersistent_flags+=("--limit")
    local_nonpersistent_flags+=("--limit=")
    flags+=("--name=")
    two_word_flags+=("--name")
    local_nonpersistent_flags+=("--name")
    local_nonpersistent_flags+=("--name=")
    flags+=("--skip=")
    two_word_flags+=("--skip")
    local_nonpersistent_flags+=("--skip")
    local_nonpersistent_flags+=("--skip=")
    flags+=("--type=")
    two_word_flags+=("--type")
    local_nonpersistent_flags+=("--type")
    local_nonpersistent_flags+=("--type=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_connection_update()
{
    last_command="iics_connection_update"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--from-file=")
    two_word_flags+=("--from-file")
    local_nonpersistent_flags+=("--from-file")
    local_nonpersistent_flags+=("--from-file=")
    flags+=("--id=")
    two_word_flags+=("--id")
    local_nonpersistent_flags+=("--id")
    local_nonpersistent_flags+=("--id=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_connection()
{
    last_command="iics_connection"

    command_aliases=()

    commands=()
    commands+=("create")
    commands+=("delete")
    commands+=("get")
    commands+=("list")
    commands+=("update")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_export_create()
{
    last_command="iics_export_create"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--ids=")
    two_word_flags+=("--ids")
    local_nonpersistent_flags+=("--ids")
    local_nonpersistent_flags+=("--ids=")
    flags+=("--include-deps")
    local_nonpersistent_flags+=("--include-deps")
    flags+=("--name=")
    two_word_flags+=("--name")
    local_nonpersistent_flags+=("--name")
    local_nonpersistent_flags+=("--name=")
    flags+=("--wait")
    local_nonpersistent_flags+=("--wait")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_export_download()
{
    last_command="iics_export_download"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--id=")
    two_word_flags+=("--id")
    local_nonpersistent_flags+=("--id")
    local_nonpersistent_flags+=("--id=")
    flags+=("--output-file=")
    two_word_flags+=("--output-file")
    two_word_flags+=("-f")
    local_nonpersistent_flags+=("--output-file")
    local_nonpersistent_flags+=("--output-file=")
    local_nonpersistent_flags+=("-f")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_export_run()
{
    last_command="iics_export_run"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--artifacts-file=")
    two_word_flags+=("--artifacts-file")
    local_nonpersistent_flags+=("--artifacts-file")
    local_nonpersistent_flags+=("--artifacts-file=")
    flags+=("--download-export-log=")
    two_word_flags+=("--download-export-log")
    local_nonpersistent_flags+=("--download-export-log")
    local_nonpersistent_flags+=("--download-export-log=")
    flags+=("--exclude-dependencies")
    local_nonpersistent_flags+=("--exclude-dependencies")
    flags+=("--expand-status")
    local_nonpersistent_flags+=("--expand-status")
    flags+=("--export-file-path=")
    two_word_flags+=("--export-file-path")
    local_nonpersistent_flags+=("--export-file-path")
    local_nonpersistent_flags+=("--export-file-path=")
    flags+=("--include-tags")
    local_nonpersistent_flags+=("--include-tags")
    flags+=("--max-wait-time=")
    two_word_flags+=("--max-wait-time")
    local_nonpersistent_flags+=("--max-wait-time")
    local_nonpersistent_flags+=("--max-wait-time=")
    flags+=("--name=")
    two_word_flags+=("--name")
    two_word_flags+=("-n")
    local_nonpersistent_flags+=("--name")
    local_nonpersistent_flags+=("--name=")
    local_nonpersistent_flags+=("-n")
    flags+=("--polling-interval=")
    two_word_flags+=("--polling-interval")
    local_nonpersistent_flags+=("--polling-interval")
    local_nonpersistent_flags+=("--polling-interval=")
    flags+=("--print-file-contents")
    local_nonpersistent_flags+=("--print-file-contents")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_export_start()
{
    last_command="iics_export_start"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--artifacts-file=")
    two_word_flags+=("--artifacts-file")
    local_nonpersistent_flags+=("--artifacts-file")
    local_nonpersistent_flags+=("--artifacts-file=")
    flags+=("--exclude-dependencies")
    local_nonpersistent_flags+=("--exclude-dependencies")
    flags+=("--include-tags")
    local_nonpersistent_flags+=("--include-tags")
    flags+=("--name=")
    two_word_flags+=("--name")
    two_word_flags+=("-n")
    local_nonpersistent_flags+=("--name")
    local_nonpersistent_flags+=("--name=")
    local_nonpersistent_flags+=("-n")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_export_status()
{
    last_command="iics_export_status"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--expand")
    local_nonpersistent_flags+=("--expand")
    flags+=("--id=")
    two_word_flags+=("--id")
    local_nonpersistent_flags+=("--id")
    local_nonpersistent_flags+=("--id=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_export()
{
    last_command="iics_export"

    command_aliases=()

    commands=()
    commands+=("create")
    commands+=("download")
    commands+=("run")
    commands+=("start")
    commands+=("status")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_folder_create()
{
    last_command="iics_folder_create"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--description=")
    two_word_flags+=("--description")
    local_nonpersistent_flags+=("--description")
    local_nonpersistent_flags+=("--description=")
    flags+=("--name=")
    two_word_flags+=("--name")
    local_nonpersistent_flags+=("--name")
    local_nonpersistent_flags+=("--name=")
    flags+=("--project-id=")
    two_word_flags+=("--project-id")
    local_nonpersistent_flags+=("--project-id")
    local_nonpersistent_flags+=("--project-id=")
    flags+=("--project-name=")
    two_word_flags+=("--project-name")
    local_nonpersistent_flags+=("--project-name")
    local_nonpersistent_flags+=("--project-name=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_folder_delete()
{
    last_command="iics_folder_delete"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--id=")
    two_word_flags+=("--id")
    local_nonpersistent_flags+=("--id")
    local_nonpersistent_flags+=("--id=")
    flags+=("--yes")
    flags+=("-y")
    local_nonpersistent_flags+=("--yes")
    local_nonpersistent_flags+=("-y")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_folder_update()
{
    last_command="iics_folder_update"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--description=")
    two_word_flags+=("--description")
    local_nonpersistent_flags+=("--description")
    local_nonpersistent_flags+=("--description=")
    flags+=("--folder-name=")
    two_word_flags+=("--folder-name")
    local_nonpersistent_flags+=("--folder-name")
    local_nonpersistent_flags+=("--folder-name=")
    flags+=("--id=")
    two_word_flags+=("--id")
    local_nonpersistent_flags+=("--id")
    local_nonpersistent_flags+=("--id=")
    flags+=("--name=")
    two_word_flags+=("--name")
    local_nonpersistent_flags+=("--name")
    local_nonpersistent_flags+=("--name=")
    flags+=("--project-id=")
    two_word_flags+=("--project-id")
    local_nonpersistent_flags+=("--project-id")
    local_nonpersistent_flags+=("--project-id=")
    flags+=("--project-name=")
    two_word_flags+=("--project-name")
    local_nonpersistent_flags+=("--project-name")
    local_nonpersistent_flags+=("--project-name=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_folder()
{
    last_command="iics_folder"

    command_aliases=()

    commands=()
    commands+=("create")
    commands+=("delete")
    commands+=("update")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_help()
{
    last_command="iics_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_iics_import_download-log()
{
    last_command="iics_import_download-log"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--id=")
    two_word_flags+=("--id")
    local_nonpersistent_flags+=("--id")
    local_nonpersistent_flags+=("--id=")
    flags+=("--log-name=")
    two_word_flags+=("--log-name")
    local_nonpersistent_flags+=("--log-name")
    local_nonpersistent_flags+=("--log-name=")
    flags+=("--log-path=")
    two_word_flags+=("--log-path")
    local_nonpersistent_flags+=("--log-path")
    local_nonpersistent_flags+=("--log-path=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_import_run()
{
    last_command="iics_import_run"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--conflict-resolution=")
    two_word_flags+=("--conflict-resolution")
    local_nonpersistent_flags+=("--conflict-resolution")
    local_nonpersistent_flags+=("--conflict-resolution=")
    flags+=("--detailed-polling")
    local_nonpersistent_flags+=("--detailed-polling")
    flags+=("--expand")
    local_nonpersistent_flags+=("--expand")
    flags+=("--from-file=")
    two_word_flags+=("--from-file")
    local_nonpersistent_flags+=("--from-file")
    local_nonpersistent_flags+=("--from-file=")
    flags+=("--max-wait-time=")
    two_word_flags+=("--max-wait-time")
    local_nonpersistent_flags+=("--max-wait-time")
    local_nonpersistent_flags+=("--max-wait-time=")
    flags+=("--name=")
    two_word_flags+=("--name")
    local_nonpersistent_flags+=("--name")
    local_nonpersistent_flags+=("--name=")
    flags+=("--object-status-fields=")
    two_word_flags+=("--object-status-fields")
    local_nonpersistent_flags+=("--object-status-fields")
    local_nonpersistent_flags+=("--object-status-fields=")
    flags+=("--polling-interval=")
    two_word_flags+=("--polling-interval")
    local_nonpersistent_flags+=("--polling-interval")
    local_nonpersistent_flags+=("--polling-interval=")
    flags+=("--print-import-log")
    local_nonpersistent_flags+=("--print-import-log")
    flags+=("--zip-file=")
    two_word_flags+=("--zip-file")
    two_word_flags+=("-z")
    local_nonpersistent_flags+=("--zip-file")
    local_nonpersistent_flags+=("--zip-file=")
    local_nonpersistent_flags+=("-z")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_import_start()
{
    last_command="iics_import_start"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--conflict-resolution=")
    two_word_flags+=("--conflict-resolution")
    local_nonpersistent_flags+=("--conflict-resolution")
    local_nonpersistent_flags+=("--conflict-resolution=")
    flags+=("--from-file=")
    two_word_flags+=("--from-file")
    local_nonpersistent_flags+=("--from-file")
    local_nonpersistent_flags+=("--from-file=")
    flags+=("--id=")
    two_word_flags+=("--id")
    local_nonpersistent_flags+=("--id")
    local_nonpersistent_flags+=("--id=")
    flags+=("--name=")
    two_word_flags+=("--name")
    local_nonpersistent_flags+=("--name")
    local_nonpersistent_flags+=("--name=")
    flags+=("--wait")
    local_nonpersistent_flags+=("--wait")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_import_status()
{
    last_command="iics_import_status"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--expand")
    local_nonpersistent_flags+=("--expand")
    flags+=("--id=")
    two_word_flags+=("--id")
    local_nonpersistent_flags+=("--id")
    local_nonpersistent_flags+=("--id=")
    flags+=("--object-status-fields=")
    two_word_flags+=("--object-status-fields")
    local_nonpersistent_flags+=("--object-status-fields")
    local_nonpersistent_flags+=("--object-status-fields=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_import_upload()
{
    last_command="iics_import_upload"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--file=")
    two_word_flags+=("--file")
    two_word_flags+=("-f")
    local_nonpersistent_flags+=("--file")
    local_nonpersistent_flags+=("--file=")
    local_nonpersistent_flags+=("-f")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_import()
{
    last_command="iics_import"

    command_aliases=()

    commands=()
    commands+=("download-log")
    commands+=("run")
    commands+=("start")
    commands+=("status")
    commands+=("upload")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_login()
{
    last_command="iics_login"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_logout()
{
    last_command="iics_logout"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_lookup()
{
    last_command="iics_lookup"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--id=")
    two_word_flags+=("--id")
    local_nonpersistent_flags+=("--id")
    local_nonpersistent_flags+=("--id=")
    flags+=("--path=")
    two_word_flags+=("--path")
    local_nonpersistent_flags+=("--path")
    local_nonpersistent_flags+=("--path=")
    flags+=("--type=")
    two_word_flags+=("--type")
    local_nonpersistent_flags+=("--type")
    local_nonpersistent_flags+=("--type=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_metering_download()
{
    last_command="iics_metering_download"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--id=")
    two_word_flags+=("--id")
    local_nonpersistent_flags+=("--id")
    local_nonpersistent_flags+=("--id=")
    flags+=("--output-file=")
    two_word_flags+=("--output-file")
    two_word_flags+=("-f")
    local_nonpersistent_flags+=("--output-file")
    local_nonpersistent_flags+=("--output-file=")
    local_nonpersistent_flags+=("-f")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_metering_get()
{
    last_command="iics_metering_get"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--end=")
    two_word_flags+=("--end")
    local_nonpersistent_flags+=("--end")
    local_nonpersistent_flags+=("--end=")
    flags+=("--start=")
    two_word_flags+=("--start")
    local_nonpersistent_flags+=("--start")
    local_nonpersistent_flags+=("--start=")
    flags+=("--type=")
    two_word_flags+=("--type")
    local_nonpersistent_flags+=("--type")
    local_nonpersistent_flags+=("--type=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_metering()
{
    last_command="iics_metering"

    command_aliases=()

    commands=()
    commands+=("download")
    commands+=("get")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_objects_dependencies()
{
    last_command="iics_objects_dependencies"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--id=")
    two_word_flags+=("--id")
    local_nonpersistent_flags+=("--id")
    local_nonpersistent_flags+=("--id=")
    flags+=("--limit=")
    two_word_flags+=("--limit")
    local_nonpersistent_flags+=("--limit")
    local_nonpersistent_flags+=("--limit=")
    flags+=("--ref-type=")
    two_word_flags+=("--ref-type")
    local_nonpersistent_flags+=("--ref-type")
    local_nonpersistent_flags+=("--ref-type=")
    flags+=("--skip=")
    two_word_flags+=("--skip")
    local_nonpersistent_flags+=("--skip")
    local_nonpersistent_flags+=("--skip=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_objects_list()
{
    last_command="iics_objects_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--limit=")
    two_word_flags+=("--limit")
    local_nonpersistent_flags+=("--limit")
    local_nonpersistent_flags+=("--limit=")
    flags+=("--output-fields=")
    two_word_flags+=("--output-fields")
    local_nonpersistent_flags+=("--output-fields")
    local_nonpersistent_flags+=("--output-fields=")
    flags+=("--output-file=")
    two_word_flags+=("--output-file")
    local_nonpersistent_flags+=("--output-file")
    local_nonpersistent_flags+=("--output-file=")
    flags+=("--output-file-fields=")
    two_word_flags+=("--output-file-fields")
    local_nonpersistent_flags+=("--output-file-fields")
    local_nonpersistent_flags+=("--output-file-fields=")
    flags+=("--output-file-format=")
    two_word_flags+=("--output-file-format")
    local_nonpersistent_flags+=("--output-file-format")
    local_nonpersistent_flags+=("--output-file-format=")
    flags+=("--query=")
    two_word_flags+=("--query")
    two_word_flags+=("-q")
    local_nonpersistent_flags+=("--query")
    local_nonpersistent_flags+=("--query=")
    local_nonpersistent_flags+=("-q")
    flags+=("--skip=")
    two_word_flags+=("--skip")
    local_nonpersistent_flags+=("--skip")
    local_nonpersistent_flags+=("--skip=")
    flags+=("--type=")
    two_word_flags+=("--type")
    local_nonpersistent_flags+=("--type")
    local_nonpersistent_flags+=("--type=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_objects()
{
    last_command="iics_objects"

    command_aliases=()

    commands=()
    commands+=("dependencies")
    commands+=("list")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_package_create()
{
    last_command="iics_package_create"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--force")
    flags+=("-f")
    local_nonpersistent_flags+=("--force")
    local_nonpersistent_flags+=("-f")
    flags+=("--source=")
    two_word_flags+=("--source")
    two_word_flags+=("-s")
    local_nonpersistent_flags+=("--source")
    local_nonpersistent_flags+=("--source=")
    local_nonpersistent_flags+=("-s")
    flags+=("--target=")
    two_word_flags+=("--target")
    two_word_flags+=("-t")
    local_nonpersistent_flags+=("--target")
    local_nonpersistent_flags+=("--target=")
    local_nonpersistent_flags+=("-t")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_flag+=("--source=")
    must_have_one_flag+=("-s")
    must_have_one_flag+=("--target=")
    must_have_one_flag+=("-t")
    must_have_one_noun=()
    noun_aliases=()
}

_iics_package_dependencies()
{
    last_command="iics_package_dependencies"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--exclude=")
    two_word_flags+=("--exclude")
    two_word_flags+=("-e")
    local_nonpersistent_flags+=("--exclude")
    local_nonpersistent_flags+=("--exclude=")
    local_nonpersistent_flags+=("-e")
    flags+=("--file=")
    two_word_flags+=("--file")
    two_word_flags+=("-f")
    local_nonpersistent_flags+=("--file")
    local_nonpersistent_flags+=("--file=")
    local_nonpersistent_flags+=("-f")
    flags+=("--filter=")
    two_word_flags+=("--filter")
    local_nonpersistent_flags+=("--filter")
    local_nonpersistent_flags+=("--filter=")
    flags+=("--order-by=")
    two_word_flags+=("--order-by")
    local_nonpersistent_flags+=("--order-by")
    local_nonpersistent_flags+=("--order-by=")
    flags+=("--publish")
    local_nonpersistent_flags+=("--publish")
    flags+=("--report=")
    two_word_flags+=("--report")
    local_nonpersistent_flags+=("--report")
    local_nonpersistent_flags+=("--report=")
    flags+=("--target-profile=")
    two_word_flags+=("--target-profile")
    two_word_flags+=("-t")
    local_nonpersistent_flags+=("--target-profile")
    local_nonpersistent_flags+=("--target-profile=")
    local_nonpersistent_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    local_nonpersistent_flags+=("--workspace")
    local_nonpersistent_flags+=("--workspace=")
    local_nonpersistent_flags+=("-w")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_package_expand()
{
    last_command="iics_package_expand"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--clean")
    flags+=("-c")
    local_nonpersistent_flags+=("--clean")
    local_nonpersistent_flags+=("-c")
    flags+=("--file=")
    two_word_flags+=("--file")
    two_word_flags+=("-f")
    local_nonpersistent_flags+=("--file")
    local_nonpersistent_flags+=("--file=")
    local_nonpersistent_flags+=("-f")
    flags+=("--recursive")
    flags+=("-r")
    local_nonpersistent_flags+=("--recursive")
    local_nonpersistent_flags+=("-r")
    flags+=("--target=")
    two_word_flags+=("--target")
    two_word_flags+=("-t")
    local_nonpersistent_flags+=("--target")
    local_nonpersistent_flags+=("--target=")
    local_nonpersistent_flags+=("-t")
    flags+=("--yes")
    flags+=("-y")
    local_nonpersistent_flags+=("--yes")
    local_nonpersistent_flags+=("-y")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_flag+=("--file=")
    must_have_one_flag+=("-f")
    must_have_one_flag+=("--target=")
    must_have_one_flag+=("-t")
    must_have_one_noun=()
    noun_aliases=()
}

_iics_package()
{
    last_command="iics_package"

    command_aliases=()

    commands=()
    commands+=("create")
    commands+=("dependencies")
    commands+=("expand")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_permission_delete()
{
    last_command="iics_permission_delete"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--object-id=")
    two_word_flags+=("--object-id")
    local_nonpersistent_flags+=("--object-id")
    local_nonpersistent_flags+=("--object-id=")
    flags+=("--yes")
    flags+=("-y")
    local_nonpersistent_flags+=("--yes")
    local_nonpersistent_flags+=("-y")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_permission_get()
{
    last_command="iics_permission_get"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--object-id=")
    two_word_flags+=("--object-id")
    local_nonpersistent_flags+=("--object-id")
    local_nonpersistent_flags+=("--object-id=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_permission_set()
{
    last_command="iics_permission_set"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--from-file=")
    two_word_flags+=("--from-file")
    local_nonpersistent_flags+=("--from-file")
    local_nonpersistent_flags+=("--from-file=")
    flags+=("--object-id=")
    two_word_flags+=("--object-id")
    local_nonpersistent_flags+=("--object-id")
    local_nonpersistent_flags+=("--object-id=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_permission()
{
    last_command="iics_permission"

    command_aliases=()

    commands=()
    commands+=("delete")
    commands+=("get")
    commands+=("set")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_privilege_list()
{
    last_command="iics_privilege_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_privilege()
{
    last_command="iics_privilege"

    command_aliases=()

    commands=()
    commands+=("list")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_profile_add()
{
    last_command="iics_profile_add"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_profile_delete()
{
    last_command="iics_profile_delete"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--yes")
    flags+=("-y")
    local_nonpersistent_flags+=("--yes")
    local_nonpersistent_flags+=("-y")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_profile_edit()
{
    last_command="iics_profile_edit"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_profile_list()
{
    last_command="iics_profile_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_profile_set-default()
{
    last_command="iics_profile_set-default"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_profile_set-password()
{
    last_command="iics_profile_set-password"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_profile_show()
{
    last_command="iics_profile_show"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_profile()
{
    last_command="iics_profile"

    command_aliases=()

    commands=()
    commands+=("add")
    commands+=("delete")
    commands+=("edit")
    commands+=("list")
    commands+=("set-default")
    commands+=("set-password")
    commands+=("show")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_project_create()
{
    last_command="iics_project_create"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--description=")
    two_word_flags+=("--description")
    local_nonpersistent_flags+=("--description")
    local_nonpersistent_flags+=("--description=")
    flags+=("--name=")
    two_word_flags+=("--name")
    local_nonpersistent_flags+=("--name")
    local_nonpersistent_flags+=("--name=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_project_delete()
{
    last_command="iics_project_delete"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--id=")
    two_word_flags+=("--id")
    local_nonpersistent_flags+=("--id")
    local_nonpersistent_flags+=("--id=")
    flags+=("--yes")
    flags+=("-y")
    local_nonpersistent_flags+=("--yes")
    local_nonpersistent_flags+=("-y")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_project_update()
{
    last_command="iics_project_update"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--description=")
    two_word_flags+=("--description")
    local_nonpersistent_flags+=("--description")
    local_nonpersistent_flags+=("--description=")
    flags+=("--id=")
    two_word_flags+=("--id")
    local_nonpersistent_flags+=("--id")
    local_nonpersistent_flags+=("--id=")
    flags+=("--name=")
    two_word_flags+=("--name")
    local_nonpersistent_flags+=("--name")
    local_nonpersistent_flags+=("--name=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_project()
{
    last_command="iics_project"

    command_aliases=()

    commands=()
    commands+=("create")
    commands+=("delete")
    commands+=("update")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_publish_run()
{
    last_command="iics_publish_run"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--asset=")
    two_word_flags+=("--asset")
    local_nonpersistent_flags+=("--asset")
    local_nonpersistent_flags+=("--asset=")
    flags+=("--cai-url=")
    two_word_flags+=("--cai-url")
    local_nonpersistent_flags+=("--cai-url")
    local_nonpersistent_flags+=("--cai-url=")
    flags+=("--detailed-polling")
    local_nonpersistent_flags+=("--detailed-polling")
    flags+=("--from-file=")
    two_word_flags+=("--from-file")
    local_nonpersistent_flags+=("--from-file")
    local_nonpersistent_flags+=("--from-file=")
    flags+=("--item-fields=")
    two_word_flags+=("--item-fields")
    local_nonpersistent_flags+=("--item-fields")
    local_nonpersistent_flags+=("--item-fields=")
    flags+=("--max-wait-time=")
    two_word_flags+=("--max-wait-time")
    local_nonpersistent_flags+=("--max-wait-time")
    local_nonpersistent_flags+=("--max-wait-time=")
    flags+=("--name=")
    two_word_flags+=("--name")
    local_nonpersistent_flags+=("--name")
    local_nonpersistent_flags+=("--name=")
    flags+=("--polling-interval=")
    two_word_flags+=("--polling-interval")
    local_nonpersistent_flags+=("--polling-interval")
    local_nonpersistent_flags+=("--polling-interval=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_publish_start()
{
    last_command="iics_publish_start"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--asset=")
    two_word_flags+=("--asset")
    local_nonpersistent_flags+=("--asset")
    local_nonpersistent_flags+=("--asset=")
    flags+=("--cai-url=")
    two_word_flags+=("--cai-url")
    local_nonpersistent_flags+=("--cai-url")
    local_nonpersistent_flags+=("--cai-url=")
    flags+=("--from-file=")
    two_word_flags+=("--from-file")
    local_nonpersistent_flags+=("--from-file")
    local_nonpersistent_flags+=("--from-file=")
    flags+=("--name=")
    two_word_flags+=("--name")
    local_nonpersistent_flags+=("--name")
    local_nonpersistent_flags+=("--name=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_publish_status()
{
    last_command="iics_publish_status"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--cai-url=")
    two_word_flags+=("--cai-url")
    local_nonpersistent_flags+=("--cai-url")
    local_nonpersistent_flags+=("--cai-url=")
    flags+=("--full")
    local_nonpersistent_flags+=("--full")
    flags+=("--id=")
    two_word_flags+=("--id")
    local_nonpersistent_flags+=("--id")
    local_nonpersistent_flags+=("--id=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_publish()
{
    last_command="iics_publish"

    command_aliases=()

    commands=()
    commands+=("run")
    commands+=("start")
    commands+=("status")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_role_create()
{
    last_command="iics_role_create"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--from-file=")
    two_word_flags+=("--from-file")
    local_nonpersistent_flags+=("--from-file")
    local_nonpersistent_flags+=("--from-file=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_role_delete()
{
    last_command="iics_role_delete"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--id=")
    two_word_flags+=("--id")
    local_nonpersistent_flags+=("--id")
    local_nonpersistent_flags+=("--id=")
    flags+=("--yes")
    flags+=("-y")
    local_nonpersistent_flags+=("--yes")
    local_nonpersistent_flags+=("-y")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_role_get()
{
    last_command="iics_role_get"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--id=")
    two_word_flags+=("--id")
    local_nonpersistent_flags+=("--id")
    local_nonpersistent_flags+=("--id=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_role_list()
{
    last_command="iics_role_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--limit=")
    two_word_flags+=("--limit")
    local_nonpersistent_flags+=("--limit")
    local_nonpersistent_flags+=("--limit=")
    flags+=("--skip=")
    two_word_flags+=("--skip")
    local_nonpersistent_flags+=("--skip")
    local_nonpersistent_flags+=("--skip=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_role_update()
{
    last_command="iics_role_update"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--from-file=")
    two_word_flags+=("--from-file")
    local_nonpersistent_flags+=("--from-file")
    local_nonpersistent_flags+=("--from-file=")
    flags+=("--id=")
    two_word_flags+=("--id")
    local_nonpersistent_flags+=("--id")
    local_nonpersistent_flags+=("--id=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_role()
{
    last_command="iics_role"

    command_aliases=()

    commands=()
    commands+=("create")
    commands+=("delete")
    commands+=("get")
    commands+=("list")
    commands+=("update")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_runtime_create()
{
    last_command="iics_runtime_create"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--from-file=")
    two_word_flags+=("--from-file")
    local_nonpersistent_flags+=("--from-file")
    local_nonpersistent_flags+=("--from-file=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_runtime_get()
{
    last_command="iics_runtime_get"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--id=")
    two_word_flags+=("--id")
    local_nonpersistent_flags+=("--id")
    local_nonpersistent_flags+=("--id=")
    flags+=("--name=")
    two_word_flags+=("--name")
    local_nonpersistent_flags+=("--name")
    local_nonpersistent_flags+=("--name=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_runtime_list()
{
    last_command="iics_runtime_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--limit=")
    two_word_flags+=("--limit")
    local_nonpersistent_flags+=("--limit")
    local_nonpersistent_flags+=("--limit=")
    flags+=("--skip=")
    two_word_flags+=("--skip")
    local_nonpersistent_flags+=("--skip")
    local_nonpersistent_flags+=("--skip=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_runtime_update()
{
    last_command="iics_runtime_update"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--from-file=")
    two_word_flags+=("--from-file")
    local_nonpersistent_flags+=("--from-file")
    local_nonpersistent_flags+=("--from-file=")
    flags+=("--id=")
    two_word_flags+=("--id")
    local_nonpersistent_flags+=("--id")
    local_nonpersistent_flags+=("--id=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_runtime()
{
    last_command="iics_runtime"

    command_aliases=()

    commands=()
    commands+=("create")
    commands+=("get")
    commands+=("list")
    commands+=("update")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_schedule_create()
{
    last_command="iics_schedule_create"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--from-file=")
    two_word_flags+=("--from-file")
    local_nonpersistent_flags+=("--from-file")
    local_nonpersistent_flags+=("--from-file=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_schedule_delete()
{
    last_command="iics_schedule_delete"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--id=")
    two_word_flags+=("--id")
    local_nonpersistent_flags+=("--id")
    local_nonpersistent_flags+=("--id=")
    flags+=("--yes")
    flags+=("-y")
    local_nonpersistent_flags+=("--yes")
    local_nonpersistent_flags+=("-y")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_schedule_get()
{
    last_command="iics_schedule_get"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--id=")
    two_word_flags+=("--id")
    local_nonpersistent_flags+=("--id")
    local_nonpersistent_flags+=("--id=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_schedule_list()
{
    last_command="iics_schedule_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--limit=")
    two_word_flags+=("--limit")
    local_nonpersistent_flags+=("--limit")
    local_nonpersistent_flags+=("--limit=")
    flags+=("--skip=")
    two_word_flags+=("--skip")
    local_nonpersistent_flags+=("--skip")
    local_nonpersistent_flags+=("--skip=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_schedule_update()
{
    last_command="iics_schedule_update"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--from-file=")
    two_word_flags+=("--from-file")
    local_nonpersistent_flags+=("--from-file")
    local_nonpersistent_flags+=("--from-file=")
    flags+=("--id=")
    two_word_flags+=("--id")
    local_nonpersistent_flags+=("--id")
    local_nonpersistent_flags+=("--id=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_schedule()
{
    last_command="iics_schedule"

    command_aliases=()

    commands=()
    commands+=("create")
    commands+=("delete")
    commands+=("get")
    commands+=("list")
    commands+=("update")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_securitylog_list()
{
    last_command="iics_securitylog_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--end=")
    two_word_flags+=("--end")
    local_nonpersistent_flags+=("--end")
    local_nonpersistent_flags+=("--end=")
    flags+=("--limit=")
    two_word_flags+=("--limit")
    local_nonpersistent_flags+=("--limit")
    local_nonpersistent_flags+=("--limit=")
    flags+=("--skip=")
    two_word_flags+=("--skip")
    local_nonpersistent_flags+=("--skip")
    local_nonpersistent_flags+=("--skip=")
    flags+=("--start=")
    two_word_flags+=("--start")
    local_nonpersistent_flags+=("--start")
    local_nonpersistent_flags+=("--start=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_securitylog()
{
    last_command="iics_securitylog"

    command_aliases=()

    commands=()
    commands+=("list")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_sourcecontrol_checkin()
{
    last_command="iics_sourcecontrol_checkin"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--comment=")
    two_word_flags+=("--comment")
    local_nonpersistent_flags+=("--comment")
    local_nonpersistent_flags+=("--comment=")
    flags+=("--object-id=")
    two_word_flags+=("--object-id")
    local_nonpersistent_flags+=("--object-id")
    local_nonpersistent_flags+=("--object-id=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_sourcecontrol_checkout()
{
    last_command="iics_sourcecontrol_checkout"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--object-id=")
    two_word_flags+=("--object-id")
    local_nonpersistent_flags+=("--object-id")
    local_nonpersistent_flags+=("--object-id=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_sourcecontrol_commit()
{
    last_command="iics_sourcecontrol_commit"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--comment=")
    two_word_flags+=("--comment")
    local_nonpersistent_flags+=("--comment")
    local_nonpersistent_flags+=("--comment=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_sourcecontrol_pull()
{
    last_command="iics_sourcecontrol_pull"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_sourcecontrol()
{
    last_command="iics_sourcecontrol"

    command_aliases=()

    commands=()
    commands+=("checkin")
    commands+=("checkout")
    commands+=("commit")
    commands+=("pull")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_state_fetch()
{
    last_command="iics_state_fetch"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--object-id=")
    two_word_flags+=("--object-id")
    local_nonpersistent_flags+=("--object-id")
    local_nonpersistent_flags+=("--object-id=")
    flags+=("--output-file=")
    two_word_flags+=("--output-file")
    two_word_flags+=("-f")
    local_nonpersistent_flags+=("--output-file")
    local_nonpersistent_flags+=("--output-file=")
    local_nonpersistent_flags+=("-f")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_state_load()
{
    last_command="iics_state_load"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--input-file=")
    two_word_flags+=("--input-file")
    two_word_flags+=("-f")
    local_nonpersistent_flags+=("--input-file")
    local_nonpersistent_flags+=("--input-file=")
    local_nonpersistent_flags+=("-f")
    flags+=("--object-id=")
    two_word_flags+=("--object-id")
    local_nonpersistent_flags+=("--object-id")
    local_nonpersistent_flags+=("--object-id=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_state()
{
    last_command="iics_state"

    command_aliases=()

    commands=()
    commands+=("fetch")
    commands+=("load")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_tag_assign()
{
    last_command="iics_tag_assign"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--object-id=")
    two_word_flags+=("--object-id")
    local_nonpersistent_flags+=("--object-id")
    local_nonpersistent_flags+=("--object-id=")
    flags+=("--tags=")
    two_word_flags+=("--tags")
    local_nonpersistent_flags+=("--tags")
    local_nonpersistent_flags+=("--tags=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_tag_remove()
{
    last_command="iics_tag_remove"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--object-id=")
    two_word_flags+=("--object-id")
    local_nonpersistent_flags+=("--object-id")
    local_nonpersistent_flags+=("--object-id=")
    flags+=("--tags=")
    two_word_flags+=("--tags")
    local_nonpersistent_flags+=("--tags")
    local_nonpersistent_flags+=("--tags=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_tag()
{
    last_command="iics_tag"

    command_aliases=()

    commands=()
    commands+=("assign")
    commands+=("remove")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_unpublish_run()
{
    last_command="iics_unpublish_run"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--asset=")
    two_word_flags+=("--asset")
    local_nonpersistent_flags+=("--asset")
    local_nonpersistent_flags+=("--asset=")
    flags+=("--cai-url=")
    two_word_flags+=("--cai-url")
    local_nonpersistent_flags+=("--cai-url")
    local_nonpersistent_flags+=("--cai-url=")
    flags+=("--detailed-polling")
    local_nonpersistent_flags+=("--detailed-polling")
    flags+=("--from-file=")
    two_word_flags+=("--from-file")
    local_nonpersistent_flags+=("--from-file")
    local_nonpersistent_flags+=("--from-file=")
    flags+=("--item-fields=")
    two_word_flags+=("--item-fields")
    local_nonpersistent_flags+=("--item-fields")
    local_nonpersistent_flags+=("--item-fields=")
    flags+=("--max-wait-time=")
    two_word_flags+=("--max-wait-time")
    local_nonpersistent_flags+=("--max-wait-time")
    local_nonpersistent_flags+=("--max-wait-time=")
    flags+=("--name=")
    two_word_flags+=("--name")
    local_nonpersistent_flags+=("--name")
    local_nonpersistent_flags+=("--name=")
    flags+=("--polling-interval=")
    two_word_flags+=("--polling-interval")
    local_nonpersistent_flags+=("--polling-interval")
    local_nonpersistent_flags+=("--polling-interval=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_unpublish_start()
{
    last_command="iics_unpublish_start"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--asset=")
    two_word_flags+=("--asset")
    local_nonpersistent_flags+=("--asset")
    local_nonpersistent_flags+=("--asset=")
    flags+=("--cai-url=")
    two_word_flags+=("--cai-url")
    local_nonpersistent_flags+=("--cai-url")
    local_nonpersistent_flags+=("--cai-url=")
    flags+=("--from-file=")
    two_word_flags+=("--from-file")
    local_nonpersistent_flags+=("--from-file")
    local_nonpersistent_flags+=("--from-file=")
    flags+=("--name=")
    two_word_flags+=("--name")
    local_nonpersistent_flags+=("--name")
    local_nonpersistent_flags+=("--name=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_unpublish_status()
{
    last_command="iics_unpublish_status"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--cai-url=")
    two_word_flags+=("--cai-url")
    local_nonpersistent_flags+=("--cai-url")
    local_nonpersistent_flags+=("--cai-url=")
    flags+=("--full")
    local_nonpersistent_flags+=("--full")
    flags+=("--id=")
    two_word_flags+=("--id")
    local_nonpersistent_flags+=("--id")
    local_nonpersistent_flags+=("--id=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_unpublish()
{
    last_command="iics_unpublish"

    command_aliases=()

    commands=()
    commands+=("run")
    commands+=("start")
    commands+=("status")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_user_change-password()
{
    last_command="iics_user_change-password"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--id=")
    two_word_flags+=("--id")
    local_nonpersistent_flags+=("--id")
    local_nonpersistent_flags+=("--id=")
    flags+=("--new-password=")
    two_word_flags+=("--new-password")
    local_nonpersistent_flags+=("--new-password")
    local_nonpersistent_flags+=("--new-password=")
    flags+=("--old-password=")
    two_word_flags+=("--old-password")
    local_nonpersistent_flags+=("--old-password")
    local_nonpersistent_flags+=("--old-password=")
    flags+=("--username=")
    two_word_flags+=("--username")
    local_nonpersistent_flags+=("--username")
    local_nonpersistent_flags+=("--username=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_user_create()
{
    last_command="iics_user_create"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--from-file=")
    two_word_flags+=("--from-file")
    local_nonpersistent_flags+=("--from-file")
    local_nonpersistent_flags+=("--from-file=")
    flags+=("--interactive")
    local_nonpersistent_flags+=("--interactive")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_user_delete()
{
    last_command="iics_user_delete"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--id=")
    two_word_flags+=("--id")
    local_nonpersistent_flags+=("--id")
    local_nonpersistent_flags+=("--id=")
    flags+=("--username=")
    two_word_flags+=("--username")
    local_nonpersistent_flags+=("--username")
    local_nonpersistent_flags+=("--username=")
    flags+=("--yes")
    flags+=("-y")
    local_nonpersistent_flags+=("--yes")
    local_nonpersistent_flags+=("-y")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_user_get()
{
    last_command="iics_user_get"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--fields=")
    two_word_flags+=("--fields")
    local_nonpersistent_flags+=("--fields")
    local_nonpersistent_flags+=("--fields=")
    flags+=("--id=")
    two_word_flags+=("--id")
    local_nonpersistent_flags+=("--id")
    local_nonpersistent_flags+=("--id=")
    flags+=("--username=")
    two_word_flags+=("--username")
    local_nonpersistent_flags+=("--username")
    local_nonpersistent_flags+=("--username=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_user_list()
{
    last_command="iics_user_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--limit=")
    two_word_flags+=("--limit")
    local_nonpersistent_flags+=("--limit")
    local_nonpersistent_flags+=("--limit=")
    flags+=("--skip=")
    two_word_flags+=("--skip")
    local_nonpersistent_flags+=("--skip")
    local_nonpersistent_flags+=("--skip=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_user_reset-password()
{
    last_command="iics_user_reset-password"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--id=")
    two_word_flags+=("--id")
    local_nonpersistent_flags+=("--id")
    local_nonpersistent_flags+=("--id=")
    flags+=("--new-password=")
    two_word_flags+=("--new-password")
    local_nonpersistent_flags+=("--new-password")
    local_nonpersistent_flags+=("--new-password=")
    flags+=("--security-answer=")
    two_word_flags+=("--security-answer")
    local_nonpersistent_flags+=("--security-answer")
    local_nonpersistent_flags+=("--security-answer=")
    flags+=("--username=")
    two_word_flags+=("--username")
    local_nonpersistent_flags+=("--username")
    local_nonpersistent_flags+=("--username=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_user_update()
{
    last_command="iics_user_update"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--from-file=")
    two_word_flags+=("--from-file")
    local_nonpersistent_flags+=("--from-file")
    local_nonpersistent_flags+=("--from-file=")
    flags+=("--id=")
    two_word_flags+=("--id")
    local_nonpersistent_flags+=("--id")
    local_nonpersistent_flags+=("--id=")
    flags+=("--interactive")
    local_nonpersistent_flags+=("--interactive")
    flags+=("--username=")
    two_word_flags+=("--username")
    local_nonpersistent_flags+=("--username")
    local_nonpersistent_flags+=("--username=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_user()
{
    last_command="iics_user"

    command_aliases=()

    commands=()
    commands+=("change-password")
    commands+=("create")
    commands+=("delete")
    commands+=("get")
    commands+=("list")
    commands+=("reset-password")
    commands+=("update")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_usergroup_create()
{
    last_command="iics_usergroup_create"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--from-file=")
    two_word_flags+=("--from-file")
    local_nonpersistent_flags+=("--from-file")
    local_nonpersistent_flags+=("--from-file=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_usergroup_delete()
{
    last_command="iics_usergroup_delete"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--id=")
    two_word_flags+=("--id")
    local_nonpersistent_flags+=("--id")
    local_nonpersistent_flags+=("--id=")
    flags+=("--yes")
    flags+=("-y")
    local_nonpersistent_flags+=("--yes")
    local_nonpersistent_flags+=("-y")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_usergroup_get()
{
    last_command="iics_usergroup_get"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--id=")
    two_word_flags+=("--id")
    local_nonpersistent_flags+=("--id")
    local_nonpersistent_flags+=("--id=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_usergroup_list()
{
    last_command="iics_usergroup_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--fields=")
    two_word_flags+=("--fields")
    local_nonpersistent_flags+=("--fields")
    local_nonpersistent_flags+=("--fields=")
    flags+=("--limit=")
    two_word_flags+=("--limit")
    local_nonpersistent_flags+=("--limit")
    local_nonpersistent_flags+=("--limit=")
    flags+=("--query=")
    two_word_flags+=("--query")
    two_word_flags+=("-q")
    local_nonpersistent_flags+=("--query")
    local_nonpersistent_flags+=("--query=")
    local_nonpersistent_flags+=("-q")
    flags+=("--skip=")
    two_word_flags+=("--skip")
    local_nonpersistent_flags+=("--skip")
    local_nonpersistent_flags+=("--skip=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_usergroup_update()
{
    last_command="iics_usergroup_update"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--from-file=")
    two_word_flags+=("--from-file")
    local_nonpersistent_flags+=("--from-file")
    local_nonpersistent_flags+=("--from-file=")
    flags+=("--id=")
    two_word_flags+=("--id")
    local_nonpersistent_flags+=("--id")
    local_nonpersistent_flags+=("--id=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_usergroup()
{
    last_command="iics_usergroup"

    command_aliases=()

    commands=()
    commands+=("create")
    commands+=("delete")
    commands+=("get")
    commands+=("list")
    commands+=("update")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_iics_root_command()
{
    last_command="iics"

    command_aliases=()

    commands=()
    commands+=("activitylog")
    commands+=("agent")
    commands+=("auditlog")
    commands+=("completion")
    commands+=("connection")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("conn")
        aliashash["conn"]="connection"
    fi
    commands+=("export")
    commands+=("folder")
    commands+=("help")
    commands+=("import")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("imp")
        aliashash["imp"]="import"
    fi
    commands+=("login")
    commands+=("logout")
    commands+=("lookup")
    commands+=("metering")
    commands+=("objects")
    commands+=("package")
    commands+=("permission")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("perm")
        aliashash["perm"]="permission"
    fi
    commands+=("privilege")
    commands+=("profile")
    commands+=("project")
    commands+=("publish")
    commands+=("role")
    commands+=("runtime")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("rt")
        aliashash["rt"]="runtime"
    fi
    commands+=("schedule")
    commands+=("securitylog")
    commands+=("sourcecontrol")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("sc")
        aliashash["sc"]="sourcecontrol"
    fi
    commands+=("state")
    commands+=("tag")
    commands+=("unpublish")
    commands+=("user")
    commands+=("usergroup")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("ug")
        aliashash["ug"]="usergroup"
    fi

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--no-color")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--theme=")
    two_word_flags+=("--theme")
    flags+=("--verbose")
    flags+=("-v")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

__start_iics()
{
    local cur prev words cword split
    declare -A flaghash 2>/dev/null || :
    declare -A aliashash 2>/dev/null || :
    if declare -F _init_completion >/dev/null 2>&1; then
        _init_completion -s || return
    else
        __iics_init_completion -n "=" || return
    fi

    local c=0
    local flag_parsing_disabled=
    local flags=()
    local two_word_flags=()
    local local_nonpersistent_flags=()
    local flags_with_completion=()
    local flags_completion=()
    local commands=("iics")
    local command_aliases=()
    local must_have_one_flag=()
    local must_have_one_noun=()
    local has_completion_function=""
    local last_command=""
    local nouns=()
    local noun_aliases=()

    __iics_handle_word
}

if [[ $(type -t compopt) = "builtin" ]]; then
    complete -o default -F __start_iics iics
else
    complete -o default -o nospace -F __start_iics iics
fi

# ex: ts=4 sw=4 et filetype=sh

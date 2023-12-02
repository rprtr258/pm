_pm() {
	args=("${COMP_WORDS[@]:1:$COMP_CWORD}")

	local IFS=$'\n'
	# COMPREPLY=($(GO_FLAGS_COMPLETION=1 ${COMP_WORDS[0]} "${args[@]}"))
	COMPREPLY=($(GO_FLAGS_COMPLETION=1 go run main.go "${args[@]}"))
	return 1
}

complete -F _pm pm

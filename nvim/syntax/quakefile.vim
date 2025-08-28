" Vim syntax file for Quakefile
" Language: Quakefile
" Maintainer: Quake Project

if exists("b:current_syntax")
  finish
endif

" Comments must be defined early to take precedence
syn match quakeComment "^\s*#.*$" contains=quakeTodo
syn keyword quakeTodo TODO FIXME XXX NOTE contained

" Keywords
syn keyword quakeKeyword task namespace file_namespace contained

" File namespace (top-level only)
syn match quakeFileNamespace "^\s*file_namespace\s\+\S\+" contains=quakeKeyword

" Variable assignments (top-level and in blocks)
syn match quakeVariableAssign "^\s*\w\+\s*=" contains=quakeVariableName,quakeEquals
syn match quakeVariableName "\w\+" contained
syn match quakeEquals "=" contained

" Variable references
syn match quakeVariable "\$\w\+"

" Arrow operator - match it globally in task lines
syn match quakeArrow "=>" contained

" Task declaration with body
syn region quakeTaskDecl start="^\s*task\s\+\w\+" end="}" contains=quakeKeyword,quakeTaskName,quakeArguments,quakeArrow,quakeDependencyList,quakeTaskBody,quakeBrace

" Bodyless task (dependencies only - no opening brace)
syn match quakeTaskBodyless "^\s*task\s\+\w\+\s*=>[^{]*$" contains=quakeKeyword,quakeTaskName,quakeArrow,quakeDependencyList

" Components
syn match quakeTaskName "\w\+" contained
syn match quakeBrace "[{}]" contained
" Also match standalone closing braces
syn match quakeBrace "^\s*}"

" Arguments region
syn region quakeArguments start="(" end=")" contained contains=quakeArgument,quakeComma
syn match quakeArgument "\w\+" contained

" Dependency list (just the dependency names, not the arrow)
syn match quakeDependencyList "[a-zA-Z0-9_:., -]\+" contained contains=quakeDependency,quakeComma
syn match quakeDependency "[a-zA-Z0-9_:.-]\+" contained
syn match quakeComma "," contained

" Namespace blocks
syn region quakeNamespaceBlock start="^\s*namespace\s\+\w\+\s*{" end="^\s*}" fold transparent contains=ALL

" Task bodies - region from { to }
syn region quakeTaskBody start="{" end="}" contained contains=quakeCommand,quakeComment,quakeVariableAssign,quakeVariable

" Commands in task body (not comments or closing braces)
syn match quakeCommand "^\s*[^#}].*$" contained contains=quakeSilentPrefix,quakeContinuePrefix,quakeString,quakeBacktick,quakeExpression,quakeVariable

" Special command prefixes
syn match quakeSilentPrefix "^\s*@" contained
syn match quakeContinuePrefix "^\s*-" contained

" Strings - only in command content
syn region quakeString start='"' skip='\\"' end='"' contained oneline
syn region quakeString start="'" skip="\\'" end="'" contained oneline
syn region quakeMultilineString start='"""' end='"""' contained

" Expressions
syn region quakeExpression start="{{" end="}}" contained contains=quakeExpressionInner,quakeExprOr,quakeExprDot
syn match quakeExpressionInner "[a-zA-Z0-9_]\+" contained
syn match quakeExprOr "||" contained
syn match quakeExprDot "\." contained

" Backticks (command substitution)
syn region quakeBacktick start="`" end="`" contained

" Highlighting
hi def link quakeComment Comment
hi def link quakeTodo Todo
hi def link quakeKeyword Keyword
hi def link quakeTaskName Function
hi def link quakeVariableName Identifier
hi def link quakeVariable PreProc
hi def link quakeEquals Operator
hi def link quakeArrow Special
hi def link quakeBrace Delimiter
hi def link quakeArgument Parameter
hi def link quakeDependency Type
hi def link quakeComma Delimiter
hi def link quakeString String
hi def link quakeMultilineString String
hi def link quakeExpression Special
hi def link quakeExpressionInner Identifier
hi def link quakeExprOr Operator
hi def link quakeExprDot Operator
hi def link quakeBacktick Special
hi def link quakeSilentPrefix SpecialChar
hi def link quakeContinuePrefix SpecialChar
hi def link quakeFileNamespace PreProc

let b:current_syntax = "quakefile"
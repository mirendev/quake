" Vim indent file for Quakefile
" Language: Quakefile

if exists("b:did_indent")
  finish
endif
let b:did_indent = 1

" Set indent options
setlocal expandtab
setlocal tabstop=2
setlocal shiftwidth=2
setlocal softtabstop=2

setlocal indentexpr=GetQuakefileIndent()
setlocal indentkeys=0{,0},!^F,o,O

if exists("*GetQuakefileIndent")
  finish
endif

function! GetQuakefileIndent()
  let lnum = prevnonblank(v:lnum - 1)
  
  if lnum == 0
    return 0
  endif
  
  let line = getline(lnum)
  let ind = indent(lnum)
  
  " Increase indent after opening brace (use 2 spaces)
  if line =~ '{\s*$'
    let ind = ind + 2
  endif
  
  " Decrease indent for closing brace (use 2 spaces)
  if getline(v:lnum) =~ '^\s*}'
    let ind = ind - 2
  endif
  
  return ind
endfunction
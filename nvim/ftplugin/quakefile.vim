" Vim filetype plugin for Quakefile
" Language: Quakefile

if exists("b:did_ftplugin")
  finish
endif
let b:did_ftplugin = 1

" Set tab options - use spaces with width of 2
setlocal expandtab
setlocal tabstop=2
setlocal shiftwidth=2
setlocal softtabstop=2

" Set comment format
setlocal commentstring=#\ %s
setlocal comments=:#

" Set fold method
setlocal foldmethod=syntax
setlocal foldlevel=99

" Match pairs
setlocal matchpairs+=<:>

" Set format options
setlocal formatoptions-=t
setlocal formatoptions+=croql
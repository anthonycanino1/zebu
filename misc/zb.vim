" Vim syntax file
" Language: Zebu
" Maintainer: Anthony Canino

if exists("b:current_syntax")
	finish
endif

syn keyword zbKeyword 	grammar
syn keyword zbReserved 	start
syn match 	zbPunc "[:\|;]"

" Only ' ' are considered strings in zebu
syn region zbString start="'" end="'"

" We use standard line and block comments
syn match zbComment "//.*$" 
syn region zbComment start="/\*" end="\*/"

hi def link zbKeyword 	Statement
hi def link zbReserved 	Type
hi def link zbPunc			Statement
hi def link zbComment 	Comment
hi def link zbString 		String

let b:current_syntax = "zb"


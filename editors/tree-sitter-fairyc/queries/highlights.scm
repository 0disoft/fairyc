(line_comment) @comment
(block_comment) @comment
(string) @string
(escape_sequence) @string.escape
(number) @number
(boolean) @boolean
(null) @constant.builtin
(identifier) @variable
(type_path) @type
(function_declaration name: (function_name) @function)
(foreign_function_declaration name: (identifier) @function)
(struct_declaration name: (identifier) @type)
(resource_declaration name: (identifier) @type)
(enum_declaration name: (identifier) @enum)
(enum_variant name: (identifier) @variant)
(parameter name: (identifier) @variable.parameter)
(field_declaration name: (identifier) @property)
(field_expression name: (identifier) @property)
(record_field name: (identifier) @property)
(attribute) @attribute
["module" "import" "as" "pub" "const" "struct" "resource" "enum" "cleanup"
 "fn" "unsafe" "extern" "ref" "ptr" "mut" "let" "var" "assert" "defer"
 "return" "discard" "burn" "if" "else" "while" "for" "in" "break"
 "continue" "region" "switch" "case" "state" "cell" "map" "static_assert"] @keyword
["+" "-" "*" "/" "%" "=" "+=" "-=" "*=" "/=" "%=" "==" "!=" "<" ">"
 "<=" ">=" "&&" "||" "&" "|" "^" "~" "!" "<<" ">>" "<<=" ">>=" "&=" "|=" "^="] @operator
["(" ")" "[" "]" "{" "}"] @punctuation.bracket
[";" "," "." "::" ":" "?" "@"] @punctuation.delimiter

(line_comment) @comment
(block_comment) @comment
(string) @string
(escape_sequence) @string.escape
(number) @number
(boolean) @boolean
(null) @constant.builtin
(identifier) @variable
(type_path) @type
((type_path (path . (identifier) @type.builtin .))
 (#match? @type.builtin "^(void|bool|u8|i32|i64|u32|usize|str|CStr)$"))
(call_expression function: (path (identifier) @function .))
(generic_call_expression (path (identifier) @function .))
(call_expression function: (field_expression name: (identifier) @function))
(constant_declaration name: (identifier) @constant)
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
(variant_expression (identifier) @variant)
(pattern (identifier) @variant)
((line_comment) @comment.doc (#match? @comment.doc "^///"))
["module" "import" "as" "pub" "const" "struct" "resource" "enum" "cleanup"
 "fn" "unsafe" "extern" "ref" "ptr" "mut" "let" "var" "assert" "defer"
 "return" "discard" "burn" "if" "else" "while" "for" "in" "break"
 "continue" "region" "switch" "case" "state" "cell" "map" "static_assert"] @keyword
["+" "-" "*" "/" "%" "=" "+=" "-=" "*=" "/=" "%=" "==" "!=" "<" ">"
 "<=" ">=" "&&" "||" "&" "|" "^" "~" "!" "<<" ">>" "<<=" ">>=" "&=" "|=" "^=" ".."] @operator
["(" ")" "[" "]" "{" "}"] @punctuation.bracket
[";" "," "." "::" ":" "?" "@"] @punctuation.delimiter

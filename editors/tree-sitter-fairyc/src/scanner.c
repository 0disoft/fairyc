#include "tree_sitter/parser.h"
#include <stdlib.h>

void *tree_sitter_fairyc_external_scanner_create(void) { return NULL; }
void tree_sitter_fairyc_external_scanner_destroy(void *payload) { (void)payload; }
unsigned tree_sitter_fairyc_external_scanner_serialize(void *payload, char *buffer) {
  (void)payload; (void)buffer; return 0;
}
void tree_sitter_fairyc_external_scanner_deserialize(void *payload, const char *buffer, unsigned length) {
  (void)payload; (void)buffer; (void)length;
}
bool tree_sitter_fairyc_external_scanner_scan(void *payload, TSLexer *lexer, const bool *valid) {
  (void)payload;
  if (!valid[0]) return false;
  while (lexer->lookahead == ' ' || lexer->lookahead == '\t' ||
         lexer->lookahead == '\r' || lexer->lookahead == '\n' ||
         lexer->lookahead == 0xfeff) lexer->advance(lexer, true);
  if (lexer->lookahead != '/') return false;
  lexer->advance(lexer, false);
  if (lexer->lookahead != '*') return false;
  lexer->advance(lexer, false);
  unsigned depth = 1;
  while (!lexer->eof(lexer)) {
    int32_t current = lexer->lookahead;
    lexer->advance(lexer, false);
    if (current == '/' && lexer->lookahead == '*') {
      lexer->advance(lexer, false); depth++;
    } else if (current == '*' && lexer->lookahead == '/') {
      lexer->advance(lexer, false);
      if (--depth == 0) { lexer->mark_end(lexer); lexer->result_symbol = 0; return true; }
    }
  }
  // Preserve an unfinished comment as trivia while the user is typing.
  lexer->mark_end(lexer); lexer->result_symbol = 0; return true;
}

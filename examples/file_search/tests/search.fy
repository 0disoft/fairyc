module tests::search;
import search as search;

@test
fn void empty_input() {
    let search::Scan scan = search::finish(search::feed(search::start(), b""));
    assert(scan.lines == 0);
    assert(scan.matches == 0);
}

@test
fn void match_crosses_reads() {
    let search::Scan first = search::feed(search::start(), b"prefix er");
    let search::Scan scan = search::finish(search::feed(first, b"ror\nERROR\n"));
    assert(scan.lines == 2);
    assert(scan.matches == 1);
}

@test
fn void counts_unterminated_line_once() {
    let search::Scan scan = search::finish(search::feed(search::start(), b"\nerrorerror\r\nerror"));
    assert(scan.lines == 3);
    assert(scan.matches == 2);
}

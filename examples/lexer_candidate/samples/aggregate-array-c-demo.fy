module main;
struct Tile { u8[2][3] pixels; bool dirty; }
fn Tile copy(Tile tile) { return tile; }
fn i32 main() {
    var tile=Tile{.pixels=[[1,2],[3,4],[5,6]],.dirty=false};
    let original=copy(tile);
    tile.pixels[1]=[7,8];
    tile.pixels[2][1]+={tile.dirty=true;2};
    assert(original.pixels[1][0]==3 && original.pixels[2][1]==6);
    assert(tile.pixels[1][0]==7 && tile.pixels[2][1]==8 && tile.dirty);
    return 0;
}

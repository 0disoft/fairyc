module tests::progress;
import progress as progress;

@test fn void percentage_boundaries_and_large_files() {
    assert(progress::percent(0,0)==0);
    assert(progress::percent(1,3)==33);assert(progress::percent(2,3)==66);
    assert(progress::percent(3,3)==100);assert(progress::percent(4,3)==100);
    assert(progress::percent(1,101)==0);assert(progress::percent(2,101)==1);
    assert(progress::percent(2305843009213693950,2305843009213693951)==99);
}


module tests::window_metrics;
import window_metrics as metrics;
@test fn void dpi_layout_and_navigation(){
    assert(metrics::valid_dpi(96) && metrics::valid_dpi(960));assert(!metrics::valid_dpi(95) && !metrics::valid_dpi(961));
    assert(metrics::scale(18,96)==18 && metrics::scale(18,144)==27 && metrics::scale(18,192)==36);
    assert(metrics::cap(3840,640,144)==960 && metrics::center(3840,960)==1440);
    assert(metrics::cap(400,640,192)==400);assert(metrics::key_action(13,0)==1);
    assert(metrics::key_action(9,0)==2 && metrics::key_action(9,1)==3);
    assert(metrics::key_action(27,0)==4 && metrics::key_action(13,1)==0);
}

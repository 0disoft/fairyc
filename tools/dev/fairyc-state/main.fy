module main;
import module_state_command_windows as command;

@effects(filesystem,console,environment,clock)
fn i32 main(){return command::run_process();}

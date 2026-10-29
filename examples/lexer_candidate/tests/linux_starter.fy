module tests::linux_starter;
import linux_starter as starter;
import std::bytes as bytes;
@link_name("linux_starter_mode") unsafe extern C fn void mode(u32 value);
@link_name("linux_starter_counts") unsafe extern C fn u32 counts();
@test @effects(filesystem) fn void validates_abi_errors_ownership_and_close(){
 var u32 i=0;while(i<9){unsafe{mode(i);};let result=starter::create(b"hello-fairy",b"example.hello-fairy",b"fairyc-language/20");
  if(i==0){assert(result.is_ok && unsafe{counts()}==111);}
  else{let p=switch(result){case .ok(_):{assert(false);return;}case .err(p):{p}};
   assert(p.kind==(if(i==1){1}else if(i==2){2}else if(i==3){3}else if(i==5){5}else if(i==6 || i==8){8}else{4}));
   assert(unsafe{counts()}==(if(i==1){100}else{111}));
  }i+=1;
 }
 unsafe{mode(0);};assert(!starter::create(b"../bad",b"example.good",b"fairyc-language/20").is_ok && unsafe{counts()}==0);
 assert(!starter::create(b"good",b"bad",b"fairyc-language/20").is_ok && unsafe{counts()}==0);
}

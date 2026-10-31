module tests::linux_gnu_policy;
import linux_gnu_policy as policy;
import std::bytes as bytes;

@test fn void admits_supported_gnu_identity_and_profiles(){
    let gcc=switch(policy::admit(false,b"glibc 2.39\n",b"gcc (Ubuntu) 13.3.0\nCopyright",b"13.3.0\n",b"x86_64-linux-gnu\n")){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(!gcc.clang && gcc.major==13);
    let clang=switch(policy::admit(true,b" glibc 2.41\n",b"Debian clang version 18.1.8\nTarget: x86_64-pc-linux-gnu",b"",b"x86_64-pc-linux-gnu\n")){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(clang.clang && clang.major==18);
    assert(policy::libc(b"glibc 3.0") && !policy::libc(b"glibc 2.38") && !policy::libc(b"musl 2.41"));
    assert(!policy::libc(b"glibc 2.39.1") && !policy::libc(b"glibc 2.39\0") && !policy::libc(b"glibc 999999999999.0"));
    assert(policy::timeout(1) && policy::timeout(600000) && !policy::timeout(0) && !policy::timeout(600001));
    assert(bytes::equal(policy::optimization(false),b"-O0") && bytes::equal(policy::optimization(true),b"-O2"));
}
@test fn void rejects_wrong_compiler_versions_and_targets(){
    assert(!policy::admit(false,b"glibc 2.41",b"gcc 12.2.0",b"12.2.0",b"x86_64-linux-gnu").is_ok);
    assert(!policy::admit(false,b"glibc 2.41",b"CLANG version 18.1.8",b"18",b"x86_64-linux-gnu").is_ok);
    assert(!policy::admit(true,b"glibc 2.41",b"clang version 17.0.0",b"",b"x86_64-linux-gnu").is_ok);
    assert(!policy::admit(true,b"glibc 2.41",b"notclang version 18.1.8",b"",b"x86_64-linux-gnu").is_ok);
    assert(!policy::admit(true,b"glibc 2.41",b"clang version 18.1.8git",b"",b"x86_64-linux-gnu").is_ok);
    assert(!policy::version(b"18.1.8.1").is_ok && !policy::version(b"18.").is_ok && !policy::version(b"+18").is_ok);
    assert(!policy::version(b"2147483648").is_ok && !policy::version(b"18.99999999999").is_ok);
    assert(!policy::target(b"x86_64-linux-musl") && !policy::target(b"x86_64-linux-gnux32") && !policy::target(b"aarch64-linux-gnu"));
    assert(!policy::target(b"x86_64-linux-gnu extra") && !policy::target(b"x86_64-linux-gnu\0") && !policy::target(b"x86_64-pc\n-linux-gnu"));
}

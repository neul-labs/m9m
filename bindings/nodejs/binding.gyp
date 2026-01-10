{
  "targets": [
    {
      "target_name": "m9m",
      "cflags!": ["-fno-exceptions"],
      "cflags_cc!": ["-fno-exceptions"],
      "sources": ["src/native/m9m.cc"],
      "include_dirs": [
        "<!@(node -p \"require('node-addon-api').include\")",
        "../../build"
      ],
      "libraries": [
        "-L<(module_root_dir)/../../build",
        "-L<(module_root_dir)/lib",
        "-lm9m"
      ],
      "defines": ["NAPI_DISABLE_CPP_EXCEPTIONS"],
      "conditions": [
        [
          "OS==\"mac\"",
          {
            "xcode_settings": {
              "GCC_ENABLE_CPP_EXCEPTIONS": "YES",
              "CLANG_CXX_LIBRARY": "libc++",
              "MACOSX_DEPLOYMENT_TARGET": "10.15"
            },
            "libraries": [
              "-Wl,-rpath,@loader_path/../../build",
              "-Wl,-rpath,@loader_path/../lib"
            ]
          }
        ],
        [
          "OS==\"linux\"",
          {
            "libraries": [
              "-Wl,-rpath,$ORIGIN/../../build",
              "-Wl,-rpath,$ORIGIN/../lib"
            ]
          }
        ]
      ]
    }
  ]
}

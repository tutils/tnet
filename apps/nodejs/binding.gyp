{
  "targets": [
    {
      "target_name": "tnet",
      "sources": ["native/addon.cpp"],
      "include_dirs": [
        "<!@(node -p \"require('node-addon-api').include\")",
        "<!(node -p \"path.resolve(__dirname, 'native/build')\")"
      ],
      "dependencies": ["<!(node -p \"require('node-addon-api').gyp\")"],
      "libraries": ["<!(node -p \"path.resolve(__dirname, 'native/build/libtnet.a')\")"],
      "cflags": ["-Wall", "-Wextra"],
      "defines": ["NODE_ADDON_API_DISABLE_CPP_EXCEPTIONS"],
      "conditions": [
        ["OS=='mac'", {
          "cflags": ["-mmacosx-version-min=10.13"],
          "xcode_settings": {
            "MACOSX_DEPLOYMENT_TARGET": "10.13",
            "LIBRARY_SEARCH_PATHS": ["<!(node -p \"path.resolve(__dirname, 'native/build')\")"]
          }
        }],
        ["OS=='win'", {
          "cflags": ["-std=c++11", "-Wno-deprecated-declarations"],
          "msvs_settings": {
            "VCCLCompilerTool": {"ExceptionHandling": 1},
            "VCLibrarianTool": {
              "AdditionalLibraryDirectories": "<!(node -p \"path.resolve(__dirname, 'native/build').replace(/\\\\/g, '/')\");"
            }
          }
        }]
      ]
    }
  ]
}
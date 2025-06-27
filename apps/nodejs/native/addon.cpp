#include <napi.h>
#include <vector>
#include "libtnet.h"

void RunCmdWrapped(const Napi::CallbackInfo& info) {
  Napi::Env env = info.Env();
  
  // 验证参数
  if (info.Length() <= 0 || !info[0].IsArray()) {
    Napi::TypeError::New(env, "Array of strings expected").ThrowAsJavaScriptException();
    return;
  }

  // 获取字符串数组参数
  Napi::Array arr = info[0].As<Napi::Array>();
  size_t length = arr.Length();
  
  // 准备 C 字符串数组
  std::vector<char*> cArray;
  for (size_t i = 0; i < length; i++) {
    Napi::Value val = arr[i];
    if (!val.IsString()) {
      Napi::TypeError::New(env, "Array must contain only strings").ThrowAsJavaScriptException();
      return;
    }
    
    std::string str = val.As<Napi::String>();
    char* cstr = new char[str.length() + 1];
    std::strcpy(cstr, str.c_str());
    cArray.push_back(cstr);
  }

  // 调用 Go 函数
  RunCmd(cArray.data(), static_cast<int>(length));
  
  // 清理内存
  for (auto ptr : cArray) delete[] ptr;
}

Napi::Object Init(Napi::Env env, Napi::Object exports) {
  exports.Set(Napi::String::New(env, "runCmd"), 
              Napi::Function::New(env, RunCmdWrapped));
  return exports;
}

NODE_API_MODULE(tnet, Init)
const tnet = require('..'); // 引用主模块

async function testRunCmd() {
  try {
    await tnet.runCmd("agent", "--help");
  } catch (err) {
    console.error("测试失败:", err);
  }
}

// 运行测试
testRunCmd();
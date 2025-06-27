const tnet = require('../build/Release/tnet.node')

module.exports = {
  /**
   * 运行tnet
   * @param args - 命令行数组
   * @returns 一个 Promise 对象，表示命令的执行结果
   */
  runCmd: function(...args) {
    return new Promise((resolve, reject) => {
      try {
        tnet.runCmd(args.map(arg => String(arg)));
        resolve();
      } catch (err) {
        reject(err);
      }
    });
  },
};
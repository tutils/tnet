// lib/index.d.ts

/**
 * 运行tnet
 * @param args - 命令行数组
 * @returns 一个 Promise 对象，表示命令的执行结果
 */
export function runCmd(...args: string[]): Promise<void>;
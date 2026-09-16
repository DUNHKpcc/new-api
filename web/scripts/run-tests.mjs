import { execFileSync } from 'node:child_process'
import { globSync, readFileSync } from 'node:fs'
import path from 'node:path'

const root = process.cwd()
const files = globSync('src/**/*.{test,spec}.{ts,tsx}', { cwd: root })
const bunFiles = files.filter((file) => {
  const source = readFileSync(path.join(root, file), 'utf8')
  return /(?:bun:test|node:test|bun:\x27test\x27|node:\x27test\x27)/.test(source)
})
const vitestFiles = files.filter((file) => !bunFiles.includes(file))

const run = (command, args) => execFileSync(command, args, { stdio: 'inherit', cwd: root })

if (vitestFiles.length > 0) {
  run('bunx', ['vitest', 'run', '--no-file-parallelism', '--maxWorkers=1', ...vitestFiles])
}
for (const file of bunFiles) run('bun', ['--preload', './scripts/bun-test-setup.mjs', 'test', file])

import { createRequire } from 'node:module'
import { describe, expect, it } from 'vitest'

const pluginRequire = createRequire(import.meta.resolve('@sveltejs/vite-plugin-svelte'))
const packageRoot = pluginRequire.resolve('deepmerge/package.json').replace(/package\.json$/, '')

for (const entry of ['dist/cjs.js', 'dist/umd.js']) {
  const merge = pluginRequire(packageRoot + entry)
  describe(`deepmerge ${entry}`, () => {
    const poisoned = () => JSON.parse('{"__proto__":{"inheritedFlag":true}}')
    function safe(result: Record<string, unknown>) {
      expect(Object.getPrototypeOf(result)).toBe(Object.prototype)
      expect(result.inheritedFlag).toBeUndefined()
      expect(({} as Record<string, unknown>).inheritedFlag).toBeUndefined()
    }

    it('keeps target and null-prototype source keys inert', () => {
      for (const [target, source] of [
        [poisoned(), {}],
        [Object.create(null), poisoned()],
        [poisoned(), poisoned()],
      ]) {
        const result = merge(target, source)
        safe(result)
        expect(Object.hasOwn(result, '__proto__')).toBe(true)
        expect(result.__proto__).toEqual({ inheritedFlag: true })
      }
    })

    it('protects recursive branches and arrays', () => {
      const result = merge({ nested: poisoned(), list: [poisoned()] }, { nested: {}, list: [] })
      safe(result.nested)
      safe(result.list[0])
      const all = merge.all([{}, { nested: poisoned() }, { nested: {} }])
      safe(all.nested)
    })

    it('preserves ordinary merge, symbols and Svelte array replacement', () => {
      const symbol = Symbol('option')
      const result = merge({ nested: { a: 1 }, list: [1], [symbol]: { a: 1 } },
        { nested: { b: 2 }, list: [2], [symbol]: { b: 2 } })
      expect(result.nested).toEqual({ a: 1, b: 2 })
      expect(result.list).toEqual([1, 2])
      expect(result[symbol]).toEqual({ a: 1, b: 2 })
      expect(merge({ constructor: { a: 1 }, prototype: { x: 1 } },
        { constructor: { b: 2 }, prototype: { y: 2 } })).toEqual({
        constructor: { a: 1, b: 2 }, prototype: { x: 1, y: 2 },
      })
      expect(merge({ preprocess: [1] }, { preprocess: [2] }, {
        arrayMerge: (target: unknown[], source: unknown[]) => source ?? target,
      }).preprocess).toEqual([2])
    })
  })
}

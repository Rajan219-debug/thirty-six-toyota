import {ref, watch, type Ref} from 'vue'

// Animates a number from its previous value to the source value with an
// easeOutCubic ramp. Honours prefers-reduced-motion (jumps straight to the
// value). No dependency — just requestAnimationFrame.
export function useCountUp(getter: () => number, duration = 900): Ref<number> {
	const display = ref(0)
	const reduce = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches ?? false
	let raf = 0

	watch(getter, to => {
		if (reduce || duration <= 0) {
			display.value = to
			return
		}
		cancelAnimationFrame(raf)
		const from = display.value
		const start = performance.now()
		const step = (now: number) => {
			const p = Math.min((now - start) / duration, 1)
			const eased = 1 - Math.pow(1 - p, 3)
			display.value = from + (to - from) * eased
			if (p < 1) {
				raf = requestAnimationFrame(step)
			} else {
				display.value = to
			}
		}
		raf = requestAnimationFrame(step)
	}, {immediate: true})

	return display
}

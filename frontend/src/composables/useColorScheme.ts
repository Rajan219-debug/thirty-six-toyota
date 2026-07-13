import {computed, watch, readonly} from 'vue'
import {createSharedComposable, tryOnMounted, usePreferredColorScheme} from '@vueuse/core'
import {useAuthStore} from '@/stores/auth'

const CLASS_DARK = 'dark'
const CLASS_LIGHT = 'light'

// This is built upon the vueuse useDark
// Main differences:
// - usePreferredColorScheme
// - doesn't allow setting via the `isDark` ref.
// - instead the store is exposed
// - value is synced via `createSharedComposable`
// https://github.com/vueuse/vueuse/blob/main/packages/core/useDark/index.ts
export const useColorScheme = createSharedComposable(() => {
	const authStore = useAuthStore()
	const store = computed(() => authStore.settings.frontendSettings.colorSchema)
	const preferredColorScheme = usePreferredColorScheme()

	const isDark = computed<boolean>(() => {
		if (store.value === 'auto') {
			return preferredColorScheme.value === 'dark'
		}
		return store.value === 'dark'
	})

	function onChanged(v: boolean) {
		const el = window?.document.querySelector('html')
		el?.classList.toggle(CLASS_DARK, v)
		el?.classList.toggle(CLASS_LIGHT, !v)
	}

	watch(isDark, onChanged, { flush: 'post' })

	tryOnMounted(() => onChanged(isDark.value))

	return {
		store,
		isDark: readonly(isDark),
	}
})

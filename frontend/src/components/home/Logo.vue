<script setup lang="ts">
import { computed } from 'vue'
import { useColorScheme } from '@/composables/useColorScheme'

const { isDark } = useColorScheme()

// Deployment-level logo override (window.CUSTOM_LOGO_URL) still wins when set.
const CustomLogo = computed(() => {
	const lightLogo = window.CUSTOM_LOGO_URL
	const darkLogo = window.CUSTOM_LOGO_URL_DARK

	if (!lightLogo && !darkLogo) return ''
	if (!darkLogo) return lightLogo
	if (!lightLogo) return darkLogo

	return isDark.value ? darkLogo : lightLogo
})
</script>

<template>
	<div>
		<img
			v-if="CustomLogo"
			:src="CustomLogo"
			alt="Thirty Six Toyota"
			class="logo"
		>
		<div
			v-else
			class="brand"
			aria-label="Thirty Six Toyota"
		>
			<span class="brand__badge">36</span>
			<span class="brand__word">
				<span class="brand__line">THIRTY SIX</span>
				<span class="brand__line brand__line--accent">TOYOTA</span>
			</span>
		</div>
	</div>
</template>

<style lang="scss" scoped>
.logo {
	color: var(--logo-text-color);
	max-inline-size: 168px;
	max-block-size: 48px;
}

.brand {
	display: inline-flex;
	align-items: center;
	gap: .55rem;
	block-size: 48px;
	user-select: none;
}

.brand__badge {
	display: grid;
	place-items: center;
	inline-size: 40px;
	block-size: 40px;
	border-radius: 9px;
	// darker than --primary: white text on the full-brightness brand blue
	// fails WCAG AA contrast (~2.4:1); this hits ~5.8:1.
	background: hsl(var(--primary-h) var(--primary-s) 30%);
	color: #ffffff;
	font-weight: 800;
	font-size: 1.2rem;
	letter-spacing: -.02em;
	box-shadow: 0 2px 6px hsla(var(--primary-hsl), .35);
}

.brand__word {
	display: flex;
	flex-direction: column;
	line-height: 1.05;
	color: var(--logo-text-color);
}

.brand__line {
	font-weight: 700;
	font-size: .82rem;
	letter-spacing: .13em;
}

.brand__line--accent {
	// same reasoning as .brand__badge above: full-brightness --primary text
	// on a light card background fails contrast, this darker shade passes.
	color: hsl(var(--primary-h) var(--primary-s) var(--primary-text-l));
}
</style>

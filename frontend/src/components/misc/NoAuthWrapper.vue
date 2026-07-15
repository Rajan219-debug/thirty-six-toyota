<template>
	<div class="no-auth-wrapper">
		<div
			class="no-auth-wrapper__aura"
			aria-hidden="true"
		/>
		<main
			id="main-content"
			class="noauth-card"
		>
			<div
				class="noauth-card__accent"
				aria-hidden="true"
			/>
			<header class="noauth-card__header">
				<div class="noauth-card__logo-plate">
					<Logo class="noauth-card__logo" />
				</div>
				<div class="noauth-card__rule" />
				<span class="noauth-card__kicker">Internal Operations Portal</span>
			</header>
			<div class="noauth-card__body">
				<Message v-if="motd !== ''">
					{{ motd }}
				</Message>
				<h2
					v-if="title"
					class="title"
				>
					{{ title }}
				</h2>
				<ApiConfig v-if="shouldShowApiConfig" />
				<slot />
			</div>
		</main>
		<footer class="noauth-trust">
			<span class="noauth-trust__badge">
				<Icon icon="lock" />
				Secure Internal Environment
			</span>
			<p class="noauth-trust__note">
				Authorized Toyota Personnel Only. All access is logged and monitored under corporate security protocols.
			</p>
		</footer>
		<Legal />
	</div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'

import Logo from '@/components/home/Logo.vue'
import Message from '@/components/misc/Message.vue'
import Legal from '@/components/misc/Legal.vue'
import ApiConfig from '@/components/misc/ApiConfig.vue'

import { useTitle } from '@/composables/useTitle'
import { useConfigStore } from '@/stores/config'
import { isDesktopApp } from '@/helpers/desktopAuth'

const props = withDefaults(
	defineProps<{
		showApiConfig?: boolean;
	}>(),
	{
		showApiConfig: false,
	},
)

const isDesktop = isDesktopApp()
const hasStoredApiUrl = isDesktop && localStorage.getItem('API_URL') !== null
const shouldShowApiConfig = computed(() => props.showApiConfig && (!isDesktop || hasStoredApiUrl))

const configStore = useConfigStore()
const motd = computed(() => configStore.motd)

const route = useRoute()
const { t } = useI18n({ useScope: 'global' })
const title = computed(() =>
	route.meta?.title ? t(route.meta.title as string) : '',
)
useTitle(() => title.value)
</script>

<style lang="scss" scoped>
.no-auth-wrapper {
	position: relative;
	overflow: hidden;
	min-block-size: 100vh;
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	padding: 2.5rem 1rem;
	background:
		radial-gradient(circle at 1px 1px, rgba(11, 31, 58, .05) 1px, transparent 0) 0 0 / 30px 30px,
		linear-gradient(180deg, #eef3f9 0%, #f6f8fb 45%, #eaeff6 100%);
}

// Soft brand aura glowing behind the card.
.no-auth-wrapper__aura {
	position: absolute;
	inset: 0;
	pointer-events: none;
	background:
		radial-gradient(58% 42% at 50% 8%, rgba(31, 143, 214, .20), transparent 70%),
		radial-gradient(46% 40% at 88% 92%, rgba(11, 31, 58, .12), transparent 70%);
}

.noauth-card {
	position: relative;
	z-index: 1;
	inline-size: 100%;
	max-inline-size: 440px;
	background-color: var(--white);
	border: 1px solid var(--grey-200);
	border-radius: 18px;
	box-shadow:
		0 24px 60px -22px rgba(11, 31, 58, .38),
		0 6px 18px -10px rgba(11, 31, 58, .18);
	overflow: hidden;
	animation: noauth-card-in .5s cubic-bezier(.2, .7, .2, 1) both;

	&__accent {
		block-size: 5px;
		background: linear-gradient(90deg, var(--primary), #0b1f3a);
	}

	&__header {
		padding: 2.25rem 2rem 1.5rem;
		text-align: center;
		border-block-end: 1px solid var(--grey-200);
		display: flex;
		flex-direction: column;
		align-items: center;
		background: linear-gradient(180deg, rgba(31, 143, 214, .07), transparent 85%);
	}

	&__logo-plate {
		display: grid;
		place-items: center;
		padding: .85rem 1.25rem;
		margin-block-end: 1rem;
		border-radius: 14px;
		background: var(--white);
		border: 1px solid var(--grey-200);
		box-shadow: 0 8px 20px -12px rgba(11, 31, 58, .3);
	}

	&__logo {
		margin-block-end: 0;
	}

	&__rule {
		inline-size: 3rem;
		block-size: 3px;
		border-radius: 3px;
		background: linear-gradient(90deg, var(--primary), #0b1f3a);
		margin-block-end: 1rem;
	}

	&__kicker {
		font-family: $tst-mono-font;
		font-size: .68rem;
		font-weight: 600;
		letter-spacing: .2em;
		text-transform: uppercase;
		color: var(--grey-500);
	}

	&__body {
		padding: 2rem;
	}
}

.noauth-trust {
	position: relative;
	z-index: 1;
	margin-block-start: 1.75rem;
	text-align: center;
	max-inline-size: 420px;

	&__badge {
		display: inline-flex;
		align-items: center;
		gap: .5rem;
		padding: .5rem 1.25rem;
		border-radius: 999px;
		background: var(--white);
		border: 1px solid var(--grey-200);
		box-shadow: 0 6px 16px -10px rgba(11, 31, 58, .35);
		font-family: $tst-mono-font;
		font-size: .68rem;
		font-weight: 600;
		letter-spacing: .12em;
		text-transform: uppercase;
		color: var(--grey-900);

		svg {
			color: var(--primary);
		}
	}

	&__note {
		margin-block-start: .75rem;
		font-size: .78rem;
		color: var(--grey-500);
	}
}

@keyframes noauth-card-in {
	from {
		opacity: 0;
		transform: translateY(14px);
	}

	to {
		opacity: 1;
		transform: translateY(0);
	}
}

@media (prefers-reduced-motion: reduce) {
	.noauth-card {
		animation: none;
	}
}
</style>

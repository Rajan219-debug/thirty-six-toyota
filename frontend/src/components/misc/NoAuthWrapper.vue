<template>
	<div class="no-auth-wrapper">
		<main
			id="main-content"
			class="noauth-card"
		>
			<header class="noauth-card__header">
				<Logo class="noauth-card__logo" />
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
	background-color: #f4f4f4;
	background-image: radial-gradient(circle at 1px 1px, rgba(0, 0, 0, .04) 1px, transparent 0);
	background-size: 32px 32px;
	min-block-size: 100vh;
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	padding: 2rem 1rem;
}

.noauth-card {
	inline-size: 100%;
	max-inline-size: 440px;
	background-color: var(--white);
	border: 1px solid var(--grey-200);
	border-radius: $radius;
	box-shadow: var(--shadow-lg, 0 10px 30px rgba(0, 0, 0, .1));
	overflow: hidden;

	&__header {
		padding: 2.25rem 2rem 1.5rem;
		text-align: center;
		border-block-end: 1px solid var(--grey-200);
		display: flex;
		flex-direction: column;
		align-items: center;
	}

	&__logo {
		margin-block-end: 1rem;
	}

	&__rule {
		inline-size: 3rem;
		block-size: 3px;
		background: var(--primary);
		margin-block-end: 1rem;
	}

	&__kicker {
		font-family: $tst-mono-font;
		font-size: .68rem;
		font-weight: 600;
		letter-spacing: .18em;
		text-transform: uppercase;
		color: var(--grey-500);
	}

	&__body {
		padding: 2rem;
	}
}

.noauth-trust {
	margin-block-start: 2rem;
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
</style>

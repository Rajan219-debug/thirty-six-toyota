<template>
	<div class="check-in">
		<div class="check-in__clock">
			<Icon icon="clock" />
			<span>{{ elapsed }}</span>
		</div>

		<BaseButton
			v-if="timeTrackingStore.hasActiveTimer"
			class="check-in__button check-in__button--stop"
			:disabled="isBusy"
			@click="checkOut"
		>
			{{ $t('dashboard.checkIn.checkOut') }}
		</BaseButton>
		<template v-else>
			<FormSelect
				v-if="projectOptions.length > 1"
				v-model="selectedProjectId"
				class="check-in__select"
				:options="projectOptions"
			/>
			<BaseButton
				class="check-in__button"
				:disabled="isBusy || selectedProjectId === null"
				@click="checkIn"
			>
				{{ $t('dashboard.checkIn.checkIn') }}
			</BaseButton>
		</template>
	</div>
</template>

<script setup lang="ts">
import {ref, computed, onMounted, onUnmounted} from 'vue'

import BaseButton from '@/components/base/BaseButton.vue'
import FormSelect from '@/components/input/FormSelect.vue'
import {useTimeTrackingStore} from '@/stores/timeTracking'
import {useProjectStore} from '@/stores/projects'

const timeTrackingStore = useTimeTrackingStore()
const projectStore = useProjectStore()

const projectOptions = computed(() => projectStore.notArchivedRootProjects.map(p => ({value: p.id, label: p.title})))
const selectedProjectId = ref<number | null>(null)

const now = ref(new Date())
let interval: ReturnType<typeof setInterval> | undefined

const elapsed = computed(() => {
	const timer = timeTrackingStore.activeTimer
	const seconds = timer ? Math.max(0, Math.floor((now.value.getTime() - timer.startTime.getTime()) / 1000)) : 0
	const pad = (n: number) => n.toString().padStart(2, '0')
	return `${pad(Math.floor(seconds / 3600))}:${pad(Math.floor((seconds % 3600) / 60))}:${pad(seconds % 60)}`
})

const isBusy = ref(false)

async function checkIn() {
	if (selectedProjectId.value === null || isBusy.value) return
	isBusy.value = true
	try {
		await timeTrackingStore.createEntry({startTime: new Date(), projectId: selectedProjectId.value})
	} finally {
		isBusy.value = false
	}
}

async function checkOut() {
	if (isBusy.value) return
	isBusy.value = true
	try {
		await timeTrackingStore.stopTimer()
	} finally {
		isBusy.value = false
	}
}

onMounted(() => {
	if (projectOptions.value.length > 0) {
		selectedProjectId.value = projectOptions.value[0].value
	}
	interval = setInterval(() => {
		now.value = new Date()
	}, 1000)
})

onUnmounted(() => {
	if (interval !== undefined) clearInterval(interval)
})
</script>

<style lang="scss" scoped>
.check-in {
	position: fixed;
	inset-block-end: 1.5rem;
	inset-inline-end: 1.5rem;
	z-index: 20;

	display: flex;
	flex-direction: column;
	gap: .5rem;
	padding: 1rem;
	background: var(--white);
	border: 1px solid var(--card-border-color);
	border-radius: 12px;
	box-shadow: 0 8px 24px rgba(0, 0, 0, .12);
	min-inline-size: 200px;

	&__clock {
		display: flex;
		align-items: center;
		gap: .4rem;
		font-family: $tst-mono-font;
		font-size: 1.1rem;
		font-weight: 700;
		color: var(--grey-900);
	}

	&__select {
		font-size: .8rem;
	}

	&__button {
		display: block;
		inline-size: 100%;
		text-align: center;
		padding: .5rem .75rem;
		border-radius: 8px;
		background: var(--success);
		color: #ffffff;
		font-weight: 600;
		font-size: .85rem;

		&:disabled {
			opacity: .6;
			cursor: not-allowed;
		}

		&--stop {
			background: var(--danger);
		}
	}
}
</style>

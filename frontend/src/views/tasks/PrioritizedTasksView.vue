<template>
	<div class="prioritized-tasks-view">
		<h1 class="title">
			{{ $t('task.accountability.prioritizedTitle') }}
		</h1>

		<div
			v-if="loading"
			class="loader-container is-loading"
		/>

		<div
			v-else
			class="columns prioritized-columns"
		>
			<div class="column">
				<h2 class="subtitle has-text-danger">
					{{ $t('task.accountability.doNow') }}
				</h2>
				<p
					v-if="tasks.doNow.length === 0"
					class="none"
				>
					{{ $t('task.accountability.noTasks') }}
				</p>
				<SingleTaskInProject
					v-for="t in tasks.doNow"
					:key="t.id"
					:the-task="t"
					:show-project="true"
				/>
			</div>
			<div class="column">
				<h2 class="subtitle has-text-warning">
					{{ $t('task.accountability.doSoon') }}
				</h2>
				<p
					v-if="tasks.doSoon.length === 0"
					class="none"
				>
					{{ $t('task.accountability.noTasks') }}
				</p>
				<SingleTaskInProject
					v-for="t in tasks.doSoon"
					:key="t.id"
					:the-task="t"
					:show-project="true"
				/>
			</div>
			<div class="column">
				<h2 class="subtitle">
					{{ $t('task.accountability.doLater') }}
				</h2>
				<p
					v-if="tasks.doLater.length === 0"
					class="none"
				>
					{{ $t('task.accountability.noTasks') }}
				</p>
				<SingleTaskInProject
					v-for="t in tasks.doLater"
					:key="t.id"
					:the-task="t"
					:show-project="true"
				/>
			</div>
		</div>
	</div>
</template>

<script setup lang="ts">
import {ref, onMounted} from 'vue'

import SingleTaskInProject from '@/components/tasks/partials/SingleTaskInProject.vue'
import TaskService, {type IPrioritizedTasks} from '@/services/task'

const taskService = new TaskService()
const loading = ref(true)
const tasks = ref<IPrioritizedTasks>({doNow: [], doSoon: [], doLater: []})

onMounted(async () => {
	try {
		tasks.value = await taskService.getPrioritizedTasks()
	} finally {
		loading.value = false
	}
})
</script>

<style lang="scss" scoped>
.prioritized-tasks-view {
	padding: 1rem;
}

.prioritized-columns {
	.none {
		color: var(--grey-400);
		font-style: italic;
	}
}
</style>

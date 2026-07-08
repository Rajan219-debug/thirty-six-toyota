<template>
	<div
		v-if="showAnything"
		class="task-accountability-banner"
	>
		<div
			v-if="task.status === TASK_STATUS.OVERDUE"
			class="notification is-danger"
		>
			{{ $t('task.accountability.overdueNotice') }}
		</div>

		<div
			v-if="task.status === TASK_STATUS.PENDING_CONFIRMATION && isAssignee"
			class="notification is-warning"
		>
			<p>{{ $t('task.accountability.claimedNotice', {assigner: assignerName}) }}</p>
			<XButton
				:loading="loading"
				variant="primary"
				@click="informAssigner"
			>
				{{ $t('task.accountability.informAssigner', {assigner: assignerName}) }}
			</XButton>
		</div>

		<div
			v-if="task.status === TASK_STATUS.PENDING_CONFIRMATION && isApprover"
			class="notification is-warning"
		>
			<p>{{ $t('task.accountability.approvalRequested') }}</p>
			<div class="buttons">
				<XButton
					:loading="loading"
					variant="primary"
					@click="approve"
				>
					{{ $t('task.accountability.approve') }}
				</XButton>
				<XButton
					:loading="loading"
					variant="tertiary"
					class="has-text-danger"
					@click="showRejectModal = true"
				>
					{{ $t('task.accountability.reject') }}
				</XButton>
			</div>
		</div>

		<div
			v-if="task.status === TASK_STATUS.OVERDUE && task.rejectionReason && isAssignee"
			class="notification is-warning"
		>
			<p>{{ $t('task.accountability.rejectedNotice', {reason: task.rejectionReason}) }}</p>
			<XButton
				:loading="loading"
				variant="tertiary"
				@click="dispute"
			>
				{{ $t('task.accountability.dispute') }}
			</XButton>
		</div>

		<div
			v-if="task.status === TASK_STATUS.COMPLETED_LATE && task.confirmedBy"
			class="notification is-info"
		>
			{{ $t('task.accountability.completedLateNotice', {approver: task.confirmedBy.name || task.confirmedBy.username}) }}
		</div>

		<Modal
			:enabled="showRejectModal"
			@close="showRejectModal = false"
			@submit="reject"
		>
			<template #header>
				<span>{{ $t('task.accountability.rejectModalHeader') }}</span>
			</template>
			<template #text>
				<div class="field">
					<div class="control">
						<textarea
							v-model="rejectReason"
							class="textarea"
							:placeholder="$t('task.accountability.rejectReasonPlaceholder')"
						/>
					</div>
				</div>
			</template>
		</Modal>
	</div>
</template>

<script setup lang="ts">
import {ref, computed} from 'vue'

import type {ITask} from '@/modelTypes/ITask'
import type {IProject} from '@/modelTypes/IProject'
import {TASK_STATUS} from '@/types/ITaskStatus'
import {PERMISSIONS} from '@/constants/permissions'

import XButton from '@/components/input/Button.vue'
import Modal from '@/components/misc/Modal.vue'

import TaskService from '@/services/task'
import {useAuthStore} from '@/stores/auth'
import {success} from '@/message'
import {useI18n} from 'vue-i18n'

const props = defineProps<{
	task: ITask,
	project?: IProject | null,
}>()

const emit = defineEmits<{
	'update:task': [task: ITask],
}>()

const {t} = useI18n({useScope: 'global'})
const authStore = useAuthStore()
const taskService = new TaskService()
const loading = ref(false)
const showRejectModal = ref(false)
const rejectReason = ref('')

const currentUserId = computed(() => authStore.info?.id)

const isAssignee = computed(() =>
	props.task.assignees.some(a => a.id === currentUserId.value),
)

const isApprover = computed(() => {
	if (!currentUserId.value) {
		return false
	}
	if (props.task.createdBy?.id === currentUserId.value) {
		return true
	}
	if (props.project?.fallbackApprover?.id === currentUserId.value) {
		return true
	}
	return props.project?.maxPermission === PERMISSIONS.ADMIN
})

const assignerName = computed(() => {
	const assigner = props.task.createdBy
	return assigner ? (assigner.name || assigner.username) : ''
})

const showAnything = computed(() => (
	props.task.status === TASK_STATUS.OVERDUE ||
	(props.task.status === TASK_STATUS.PENDING_CONFIRMATION && (isAssignee.value || isApprover.value)) ||
	(props.task.status === TASK_STATUS.COMPLETED_LATE && !!props.task.confirmedBy)
))

async function informAssigner() {
	loading.value = true
	try {
		const updated = await taskService.informAssigner(props.task.id)
		emit('update:task', updated)
		success({message: t('task.accountability.informedSuccess')})
	} finally {
		loading.value = false
	}
}

async function approve() {
	loading.value = true
	try {
		const updated = await taskService.approveClosure(props.task.id)
		emit('update:task', updated)
		success({message: t('task.accountability.approvedSuccess')})
	} finally {
		loading.value = false
	}
}

async function reject() {
	loading.value = true
	try {
		const updated = await taskService.rejectClosure(props.task.id, rejectReason.value)
		emit('update:task', updated)
		showRejectModal.value = false
		rejectReason.value = ''
		success({message: t('task.accountability.rejectedSuccess')})
	} finally {
		loading.value = false
	}
}

async function dispute() {
	loading.value = true
	try {
		const updated = await taskService.disputeRejection(props.task.id)
		emit('update:task', updated)
		success({message: t('task.accountability.disputedSuccess')})
	} finally {
		loading.value = false
	}
}
</script>

<style lang="scss" scoped>
.task-accountability-banner {
	.notification {
		margin-block-end: 1rem;

		.buttons {
			margin-block-start: .5rem;
		}
	}
}
</style>

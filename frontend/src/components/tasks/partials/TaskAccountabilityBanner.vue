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
			class="tst-closure"
		>
			<div class="tst-closure__head">
				<Icon icon="exclamation" />
				<span>{{ $t('task.accountability.closureHeader') }}</span>
			</div>
			<div class="tst-closure__restriction">
				<Icon :icon="['far', 'clock']" />
				<div>
					<span class="tst-closure__restriction-label">{{ $t('task.accountability.statusRestriction') }}</span>
					<p>{{ $t('task.accountability.closureRestricted') }}</p>
				</div>
			</div>
			<p class="tst-closure__instructions">
				{{ $t('task.accountability.claimedNotice', {assigner: assignerName}) }}
			</p>
			<label class="tst-closure__check">
				<input
					v-model="informChecked"
					type="checkbox"
				>
				<span>
					<span class="tst-closure__check-label">{{ $t('task.accountability.informAssigner', {assigner: assignerName}) }}</span>
					<span class="tst-closure__check-note">{{ $t('task.accountability.auditNote') }}</span>
				</span>
			</label>
			<div class="tst-closure__foot">
				<XButton
					:loading="loading"
					:disabled="!informChecked"
					variant="primary"
					@click="informAssigner"
				>
					{{ $t('task.accountability.requestClosureApproval') }}
				</XButton>
			</div>
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
const informChecked = ref(false)

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

.tst-closure {
	margin-block-end: 1rem;
	border: 1px solid var(--grey-200);
	border-radius: 12px;
	background: var(--white);
	box-shadow: 0 2px 8px rgba(0, 0, 0, .06);
	overflow: hidden;

	&__head {
		display: flex;
		align-items: center;
		gap: .6rem;
		padding: .9rem 1.25rem;
		background: var(--grey-50);
		border-block-end: 1px solid var(--grey-200);
		font-size: 1.05rem;
		font-weight: 700;
		color: var(--grey-900);

		svg {
			color: var(--danger);
		}
	}

	&__restriction {
		display: flex;
		align-items: flex-start;
		gap: .6rem;
		margin: 1.25rem 1.25rem 0;
		padding: .9rem 1rem;
		border-radius: 0 6px 6px 0;
		background: var(--grey-50);
		border-inline-start: 4px solid var(--danger);
		color: var(--grey-900);

		p {
			margin: 0;
			font-size: .85rem;
			color: var(--grey-600);
		}
	}

	&__restriction-label {
		display: block;
		font-size: .68rem;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: .04em;
		margin-block-end: .15rem;
	}

	&__instructions {
		padding: 1.1rem 1.25rem 0;
		margin: 0;
		font-size: .9rem;
		color: var(--grey-800);
	}

	&__check {
		display: flex;
		align-items: flex-start;
		gap: .75rem;
		margin: 1rem 1.25rem;
		padding: 1.1rem;
		border-radius: 8px;
		background: var(--grey-50);
		border: 1px solid var(--grey-200);
		cursor: pointer;

		input {
			margin-block-start: .2rem;
			inline-size: 1.35rem;
			block-size: 1.35rem;
			accent-color: var(--primary);
			flex-shrink: 0;
		}
	}

	&__check-label {
		display: block;
		font-weight: 600;
		color: var(--grey-900);
	}

	&__check-note {
		display: block;
		font-size: .78rem;
		color: var(--grey-500);
		margin-block-start: .2rem;
	}

	&__foot {
		display: flex;
		justify-content: flex-end;
		padding: 0 1.25rem 1.25rem;
	}
}
</style>

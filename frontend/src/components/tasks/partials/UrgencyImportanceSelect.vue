<template>
	<div class="urgency-importance-select">
		<div class="field">
			<label class="label">{{ $t('task.accountability.urgency') }}</label>
			<div class="select">
				<select
					v-model="urgency"
					:disabled="disabled || undefined"
				>
					<option
						v-for="n in LEVELS"
						:key="n"
						:value="n"
					>
						{{ n === 0 ? $t('task.accountability.levelUnset') : n }}
					</option>
				</select>
			</div>
		</div>
		<div class="field">
			<label class="label">{{ $t('task.accountability.importance') }}</label>
			<div class="select">
				<select
					v-model="importance"
					:disabled="disabled || undefined"
				>
					<option
						v-for="n in LEVELS"
						:key="n"
						:value="n"
					>
						{{ n === 0 ? $t('task.accountability.levelUnset') : n }}
					</option>
				</select>
			</div>
		</div>
	</div>
</template>

<script setup lang="ts">
const LEVELS = [0, 1, 2, 3, 4, 5]

withDefaults(defineProps<{
	disabled?: boolean
}>(), {
	disabled: false,
})

const urgency = defineModel<number>('urgency', {
	required: true,
	default: 0,
})
const importance = defineModel<number>('importance', {
	required: true,
	default: 0,
})
</script>

<style lang="scss" scoped>
.urgency-importance-select {
	display: flex;
	gap: 1rem;

	.field {
		margin-block-end: 0;
	}

	.label {
		font-weight: normal;
		color: var(--grey-400);
		font-size: .9rem;
	}
}
</style>

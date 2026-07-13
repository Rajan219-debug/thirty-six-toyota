import {ref, onMounted} from 'vue'

import TeamService from '@/services/team'
import {getDisplayName} from '@/models/user'
import type {ITeamMember} from '@/modelTypes/ITeamMember'

// No standalone "all org users" endpoint for non-admins — teams already embed
// their members, so this dedupes across every team the current user can see.
// Shared by the Users directory page and Discussion's new-chat member picker.
export function useOrgUsers() {
	const loading = ref(true)
	const users = ref<ITeamMember[]>([])

	onMounted(async () => {
		try {
			const teams = await new TeamService().getAll()
			const byId = new Map<number, ITeamMember>()
			for (const team of teams) {
				for (const member of team.members ?? []) {
					byId.set(member.id, member)
				}
			}
			users.value = [...byId.values()].sort((a, b) => getDisplayName(a).localeCompare(getDisplayName(b)))
		} finally {
			loading.value = false
		}
	})

	return {loading, users}
}

<script lang="ts">
	import type { TripPurpose } from '@travel-calendar/shared';

	export let purpose: TripPurpose;

	const purposeLabels: Record<TripPurpose, string> = {
		conference: 'Conference',
		vacation: 'Vacation',
		business: 'Business',
		family: 'Family',
		other: 'Other'
	};
</script>

<span class="trip-badge trip-badge-{purpose}">
	{purposeLabels[purpose]}
</span>

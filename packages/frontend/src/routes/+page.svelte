<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';

	// Redirect to calendar on mount
	onMount(() => {
		goto('/calendar');
	});
</script>

<div class="flex items-center justify-center min-h-screen">
	<p class="text-gray-500">Redirecting...</p>
</div>

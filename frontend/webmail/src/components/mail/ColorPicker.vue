<template>
  <div class="color-picker">
    <!-- Color Grid -->
    <div class="grid grid-cols-7 gap-1">
      <button
        v-for="color in colors"
        :key="color"
        @click="selectColor(color)"
        :class="selectedColor === color ? 'ring-2 ring-blue-500 scale-110' : 'hover:scale-110'"
        class="w-7 h-7 rounded border border-gray-300 dark:border-gray-600 transition-transform"
        :style="{ backgroundColor: color }"
        :title="color"
      ></button>
    </div>
    
    <!-- Clear Button for Background Mode -->
    <div v-if="mode === 'bg'" class="mt-2 pt-2 border-t border-gray-200 dark:border-dark-border flex justify-center">
      <button
        @click="selectColor('')"
        :class="selectedColor === '' ? 'bg-blue-50 dark:bg-blue-900/30 text-blue-700 dark:text-blue-400 border-blue-300 dark:border-blue-700' : 'text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 border-gray-300 dark:border-gray-600'"
        class="px-3 py-1 text-xs rounded border transition-colors"
      >
        None
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'

const props = defineProps<{
  initialColor?: string
  mode?: 'text' | 'bg'
}>()

const emit = defineEmits<{
  select: [color: string]
}>()

const selectedColor = ref(props.initialColor || '#000000')

// Text colors: blacks, grays, and vibrant colors
const textColors = [
  '#000000', '#333333', '#666666', '#999999', '#CCCCCC', '#E0E0E0', '#FFFFFF',
  '#FF0000', '#FF6600', '#FFCC00', '#00CC00', '#0099CC', '#0066FF',
  '#9900CC', '#FF0066', '#996633', '#006666', '#660099', '#CC6600',
  '#339933', '#003366', '#663300', '#993366', '#333300', '#003333',
]

// Background colors: pastel and light colors
const bgColors = [
  '#FFFFCC', '#CCFFCC', '#CCFFFF', '#FFCCCC', '#F0F0F0', '#FFE6CC', '#E6CCFF',
  '#FFFF99', '#CCFF99', '#99FFFF', '#FF9999', '#FFE0B2', '#D1C4E9',
  '#FFECB3', '#C8E6C9', '#B3E5FC', '#F8BBD0', '#FFF9C4', '#DCEDC8',
  '#B2EBF2', '#FFCDD2', '#FFE0B2', '#E1BEE7', '#F1F8E9', '#E0F7FA',
]

const colors = computed(() => {
  return props.mode === 'bg' ? bgColors : textColors
})

function selectColor(color: string) {
  selectedColor.value = color
  emit('select', color)
}
</script>

<style scoped>
.color-picker {
  min-width: 200px;
}
</style>

<template>
  <div class="control-bar">
    <!-- 状态一：正在录音 -->
    <div v-if="isRecording" class="controls-container">
      <div class="timer">{{ formattedTime }}</div>
      <el-button type="danger" circle size="large" @click="$emit('stop')">
        <el-icon :size="24"><VideoPause /></el-icon>
      </el-button>
    </div>

    <!-- 状态二：录音已完成，等待操作 -->
    <!-- 【重大修改】增加了 :disabled 和 :loading 属性 -->
    <div v-else-if="audioBlob" class="post-recording-actions">
      <audio :src="audioURL" controls class="preview-player" :disabled="isUploading"></audio>
      <el-button @click="$emit('reset')" :disabled="isUploading">重录</el-button>
      <el-button 
        type="success" 
        @click="$emit('upload')" 
        :loading="isUploading"
      >
        {{ isUploading ? '上传中...' : '上传录音' }}
      </el-button>
    </div>

    <!-- 状态三：初始待命状态 -->
    <div v-else class="controls-container">
      <el-button type="primary" circle size="large" @click="$emit('start')">
        <el-icon :size="24"><Microphone /></el-icon>
      </el-button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, defineProps, defineEmits } from 'vue'
import { Microphone, VideoPause } from '@element-plus/icons-vue'

// 【关键修改】接收新增的 isUploading prop
const props = defineProps({
  isRecording: Boolean,
  audioBlob: Blob,
  isUploading: Boolean
})

const emit = defineEmits(['start', 'stop', 'upload', 'reset'])

const audioURL = ref('')
const timer = ref(0)
let intervalId = null

watch(() => props.isRecording, (newVal) => {
  if (newVal) {
    timer.value = 0
    intervalId = setInterval(() => { timer.value++ }, 1000)
  } else {
    clearInterval(intervalId)
  }
})

watch(() => props.audioBlob, (newBlob) => {
  if (newBlob) {
    audioURL.value = URL.createObjectURL(newBlob)
  } else {
    // 当 audioBlob 被父组件重置为 null 时，清除 URL
    if (audioURL.value) {
      URL.revokeObjectURL(audioURL.value)
    }
    audioURL.value = ''
  }
})

const formattedTime = computed(() => {
  const minutes = Math.floor(timer.value / 60).toString().padStart(2, '0')
  const seconds = (timer.value % 60).toString().padStart(2, '0')
  return `${minutes}:${seconds}`
})
</script>

<style scoped>
/* 样式部分保持不变 */
.control-bar {
  position: fixed;
  bottom: 20px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 1001; 
  background: rgba(245, 247, 250, 0.85);
  backdrop-filter: saturate(180%) blur(15px);
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
  border-radius: 999px;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 48px; 
  padding: 8px 20px;
  border: 1px solid #e0e0e0;
  box-sizing: border-box;
  transition: opacity 0.3s, transform 0.3s;
}
.controls-container, .post-recording-actions {
  display: flex;
  align-items: center;
  gap: 15px;
}
.timer {
  font-size: 20px;
  font-family: 'Courier New', Courier, monospace;
  width: 80px;
  text-align: center;
}
.preview-player {
  height: 40px;
}
</style>
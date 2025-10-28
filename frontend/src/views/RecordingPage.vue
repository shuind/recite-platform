<template>
  <div v-if="text">
    <h1>{{ text.title }}</h1>
    <p style="white-space: pre-wrap;">{{ text.content }}</p>
    <hr/>
    <el-button @click="startRecording" :disabled="isRecording">开始录音</el-button>
    <el-button @click="stopRecording" :disabled="!isRecording">停止录音</el-button>
    <div v-if="audioURL" style="margin-top: 20px;">
      <audio :src="audioURL" controls></audio>
      <el-button @click="uploadRecording" type="success" style="margin-left: 10px;">上传录音</el-button>
    </div>
  </div>
  <div v-else>
    加载中...
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import apiClient from '@/api'
import { ElMessage } from 'element-plus'

const route = useRoute()
const router = useRouter()
const text = ref(null)
const isRecording = ref(false)
const audioBlob = ref(null)
const audioURL = ref('')
let mediaRecorder = null
let audioChunks = []

onMounted(async () => {
  const textId = route.params.id
  try {
    const response = await apiClient.get(`/texts/${textId}`)
    text.value = response.data
  } catch (error) {
    console.error('Failed to fetch text details:', error)
  }
})

const startRecording = async () => {
  const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
  mediaRecorder = new MediaRecorder(stream)
  mediaRecorder.ondataavailable = event => {
    audioChunks.push(event.data)
  }
  mediaRecorder.onstop = () => {
    audioBlob.value = new Blob(audioChunks, { type: 'audio/webm' })
    audioURL.value = URL.createObjectURL(audioBlob.value)
    audioChunks = []
  }
  mediaRecorder.start()
  isRecording.value = true
}

const stopRecording = () => {
  mediaRecorder.stop()
  isRecording.value = false
}

const uploadRecording = async () => {
  if (!audioBlob.value) return

  const formData = new FormData()
  formData.append('audio_file', audioBlob.value, 'recording.webm')
  formData.append('text_id', text.value.id)

  try {
    await apiClient.post('/recordings/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })
    ElMessage.success('上传成功！')
    router.push('/my-recordings')
  } catch (error) {
    ElMessage.error('上传失败！')
    console.error('Upload failed:', error)
  }
}
</script>
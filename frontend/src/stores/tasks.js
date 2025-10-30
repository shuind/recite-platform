// src/stores/tasks.js
import { defineStore } from 'pinia'
import api from '@/api' // 你项目的 axios 实例（src/api/index.js 默认导出）

export const useTaskStore = defineStore('task', {
  state: () => ({
    items: [],
    params: { view: 'auto', status: '', scope: 'active' }, // view: auto|manual
    
    loading: false,
  }),
  actions: {
    async fetch(p = {}) {
      this.loading = true
      try {
        this.params = { ...this.params, ...p }
        const { data } = await api.get('/tasks', { params: this.params })
        this.items = data.items || []
      } finally {
        this.loading = false
      }
    },
    async create(payload) {
      await api.post('/tasks', payload)
      await this.fetch()
    },
    async update(id, patch) {
      await api.patch(`/tasks/${id}`, patch)
      await this.fetch(this.params)
    },
    async remove(id) {
      await api.delete(`/tasks/${id}`)
      await this.fetch(this.params)
    },
    async complete(id) {
      await api.post(`/tasks/${id}/complete`)
      await this.fetch(this.params)
    },
    async snooze(id, minutes = 30) {
      await api.post(`/tasks/${id}/snooze`, { minutes })
      await this.fetch(this.params)
    },
    async reorder(ids) {
      await api.put('/tasks/reorder', { ids })
      await this.fetch({ view: 'manual' })
    },
    async undo(id) {
      await api.post(`/tasks/${id}/undo`)
      await this.fetch(this.params)
    },

  },
})

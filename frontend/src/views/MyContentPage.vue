<template>
  <!-- 这个最外层 div 现在是透明的，它的父级 .app-main-content 也是透明的，
       所以我们能看到 App.vue 中 .global-layout 的灰蓝色背景 -->
       
  <div class="my-content-page">
    
    <div class="content-layout-container"
     :class="{ 'with-right': !isRightPanelCollapsed }">
      
      <!-- ============================================= -->
      <!--      折叠/展开按钮 (已移至外部并优化)         -->
      <!-- ============================================= -->
      <div 
        class="left-panel-toggle" 
        :class="{ collapsed: isLeftPanelCollapsed }"
        @click="toggleLeftPanel"
      >
        <el-icon><ArrowLeft v-if="!isLeftPanelCollapsed" /><ArrowRight v-else /></el-icon>
      </div>
      
      <!-- ============================================= -->
      <!--            左侧导航面板 (Left Panel)           -->
      <!-- ============================================= -->
      <!-- 【关键修改】这个面板现在是透明的 -->
      <div class="left-panel" :class="{ collapsed: isLeftPanelCollapsed }">
        <div class="panel-header">
          <h3>我的内容</h3>
        </div>
        <div class="panel-content">
          <div class="panel-search">
            <el-input
              v-model="searchQuery"
              placeholder="搜索我的内容..."
              :prefix-icon="Search"
              @keyup.enter="performSearch"
              @clear="cancelSearch"
              clearable
            />
          </div>
          <div class="panel-actions">
            <el-button :icon="FolderAdd" @click="handleCreateNode('folder', selectedNode)">新建文件夹</el-button>
            <el-button :icon="DocumentAdd" @click="handleCreateNode('text', selectedNode)">新建文件</el-button>
          </div>
          <div class="panel-tree" v-loading="treeLoading">
            <el-scrollbar class="tree-scrollbar">
              <el-tree
                v-if="!isSearching"
                ref="treeRef"
                :props="treeProps"
                :load="loadNode"
                lazy
                node-key="id"
                highlight-current
                @node-click="handleNodeClick"
                @node-contextmenu="handleNodeContextMenu"
                class="custom-tree"
                draggable
                :allow-drop="allowDrop"
                @node-drop="handleNodeDrop"
              >
                <template #default="{ node, data }">
                  <span class="custom-tree-node">
                    <el-icon v-if="data.node_type === 'folder'"><Folder /></el-icon>
                    <el-icon v-else><Document /></el-icon>
                    <span>{{ node.label }}</span>
                  </span>
                </template>
              </el-tree>
              
              <div v-else class="search-results">
                <div v-if="!searchResults.length" class="search-empty">无结果</div>
                <div v-for="item in searchResults" :key="item.id" class="search-result-item" @click="handleNodeClick(item)">
                  <el-icon><Document /></el-icon>
                  <span>{{ item.title }}</span>
                </div>
              </div>
            </el-scrollbar>
          </div>
        </div>
      </div>

      <!-- ============================================= -->
      <!--      中间主内容区 (Center Panel)               -->
      <!-- ============================================= -->
      <!-- 【关键修改】这是唯一一个白色卡片 -->
      <div class="center-panel">
        <div v-if="selectedNode && selectedNode.node_type === 'text'" class="content-view-wrapper">
          <div class="center-panel-header">
            <h2 class="filename">{{ selectedNode.title }}</h2>
            <div class="actions-group" v-if="canManageContent">
              <span class="save-status">{{ saveStatus }}</span>
              <el-button @click="saveContent">保存内容</el-button>
              <el-button @click="handleRenameNode(selectedNode)">重命名</el-button>
              <el-button type="danger" @click="handleDeleteNode(selectedNode)">删除</el-button>
              <el-button type="primary" @click="openPublishDialog" v-if="isOwnerOfAnyDomain">发布到圈子...</el-button>
            </div>
          </div>
          <div class="editor-area">
            <el-input
              v-model="selectedNode.content"
              type="textarea"
              placeholder="在这里输入或粘贴你的朗读材料..."
              @input="onContentChange"
            />
          </div>
          <div class="practice-area"> 
              <div class="recording-controls">
                  <!-- 【关键修改】绑定所有的 props 和 events -->
                  <RecordingControlBar 
                      :is-recording="isRecording"
                      :audio-blob="audioBlob"
                      :is-uploading="isUploading"
                      @start="startRecording"
                      @stop="stopRecording"
                      @upload="handleUploadRequest"
                      @reset="resetRecording"
                  />
              </div>
          </div>
          <div class="recording-section">
            
            <h3 style="margin-top: 5px;">历史录音</h3>
            <div class="history-list">
              <el-table :data="nodeRecordings" stripe height="100%"> 
                <el-table-column label="录音标题" prop="title" width="200">
                  <template #default="scope">
                    <span>{{ scope.row.title || '未命名' }}</span>
                  </template>
                </el-table-column>
                <!-- 【已修复】CreatedAt -> created_at -->
                <el-table-column label="录音时间" prop="created_at" width="200">
                  <template #default="scope">
                    <span>{{ new Date(scope.row.created_at).toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai' }) }}</span>
                  </template>
                </el-table-column>
                <!-- 【已修复】Status -> status -->
                <el-table-column label="状态" prop="status" width="120" />
                
                <el-table-column label="操作" >
                  <template #default="scope">
                    <!-- 【已修复】Status -> status, AudioURL -> audio_url -->
                    <div v-if="scope.row.status === 'completed'" class="recording-actions">
                      <audio :src="scope.row.audio_url" controls></audio>
                      <div class="action-buttons">
                        <el-button link type="primary" @click="handleRenameRecording(scope.row)">重命名</el-button>
                        <el-button link type="danger" @click="handleDeleteRecording(scope.row)">删除</el-button>
                      </div>
                    </div>
                    <span v-else>处理中...</span>
                  </template>
                </el-table-column>
                
                <el-table-column label="识别结果" width="150" align="center">
                  <template #default="scope">
                    <!-- 状态一：识别成功 (逻辑不变) -->
                    <el-popover
                      v-if="scope.row.ai_status === 'completed'"
                      placement="top-start"
                      title="语音识别结果"
                      :width="400"
                      trigger="click"
                    >
                      <template #reference>
                        <el-button type="success" plain size="small">查看结果</el-button>
                      </template>
                      <div class="recognition-text-popover">
                        <p>{{ scope.row.recognized_text }}</p>
                      </div>
                    </el-popover>

                    <!-- 状态二：正在处理 (有加载动画) -->
                    <div v-else-if="scope.row.ai_status === 'processing'" class="status-indicator">
                      <el-icon class="is-loading"><Loading /></el-icon>
                      <span>正在识别...</span>
                    </div>

                    <!-- 状态三：【新增】在队列中等待 (无动画，不可点) -->
                    <div v-else-if="scope.row.ai_status === 'pending'" class="status-indicator">
                      <span>队列中...</span>
                    </div>

                    <!-- 状态四：识别失败 (逻辑不变) -->
                    <div v-else-if="scope.row.ai_status === 'failed'">
                      <el-tag type="danger" size="small">识别失败</el-tag>
                      <el-button link type="primary" size="small" @click="retryTranscribe(scope.row)" style="margin-left: 5px;">重试</el-button>
                    </div>

                    <!-- 状态五：初始状态 (null, "", undefined 等，可点击) -->
                    <div v-else>
                      <el-button type="primary" plain size="small" @click="retryTranscribe(scope.row)">点击识别</el-button>
                    </div>
                  </template>
                </el-table-column>
              </el-table>
            </div>
          </div>
        </div>
        <div v-else-if="selectedNode && selectedNode.node_type === 'folder'" class="content-view-wrapper">
          <div class="center-panel-header">
            <h2 class="filename"><el-icon><Folder /></el-icon> {{ selectedNode.title }}</h2>
            <div class="actions-group">
              <el-button @click="handleRenameNode(selectedNode)">重命名</el-button>
              <el-button type="danger" @click="handleDeleteNode(selectedNode)">删除</el-button>
            </div>
          </div>
          <el-empty :description="`这是一个文件夹。你可以在其中新建文件，或从左侧选择一个文件进行编辑。`" />
        </div>
        <el-empty v-else description="请从左侧选择一个文件开始创作或练习" />
      </div>

      <!-- ============================================= -->
      <!--      右侧 AI 助手 (Right Panel)               -->
      <!-- ============================================= -->
      <!-- 【关键修改】这个面板也是透明的 -->
      <!-- MyContentPage.vue -> .right-panel -->
      <div class="right-panel" :class="{ collapsed: isRightPanelCollapsed }">
        <div class="panel-header">
          <h3>AI 助手</h3>
        </div>
        <!-- 【关键修改】替换 el-empty -->
        <div class="panel-content ai-assistant">
          <!-- 聊天记录显示区域 -->
          <div class="chat-history">
            <el-scrollbar ref="chatScrollbarRef">
              <div v-for="(message, index) in chatMessages" :key="index" class="chat-message" :class="message.role">
                <div class="message-bubble">{{ message.content }}</div>
              </div>
              <div v-if="isAiThinking" class="chat-message assistant">
                  <div class="message-bubble loading-bubble">...</div>
              </div>
            </el-scrollbar>
          </div>
          <!-- 输入区域 -->
          <div class="chat-input">
            <el-input
              v-model="userInput"
              placeholder="输入你的问题..."
              @keyup.enter="sendMessage"
              :disabled="isAiThinking"
            >
              <template #append>
                <el-button @click="sendMessage" :disabled="isAiThinking">发送</el-button>
              </template>
            </el-input>
          </div>
        </div>
      </div>
    </div>
    
    <el-button class="right-panel-toggle" circle @click="toggleRightPanel" :icon="MoreFilled" title="AI 助手" />

    <div v-if="contextMenu.visible" :style="contextMenu.style" class="context-menu" @click="contextMenu.visible = false">
      <div class="menu-item" @click="handleRenameNode(contextMenu.node)">重命名</div>
      <div class="menu-item" @click="handleDeleteNode(contextMenu.node)">删除</div>
      <div class="menu-separator" v-if="isOwnerOfAnyDomain"></div>
      <div class="menu-item" v-if="isOwnerOfAnyDomain" @click="openPublishDialogFromContextMenu">发布到圈子...</div>
      <div v-if="contextMenu.node?.data.node_type === 'folder'" class="menu-separator"></div>
      <div v-if="contextMenu.node?.data.node_type === 'folder'" class="menu-item" @click="handleCreateNode('folder', contextMenu.node)">新建子文件夹</div>
      <div v-if="contextMenu.node?.data.node_type === 'folder'" class="menu-item" @click="handleCreateNode('text', contextMenu.node)">新建子文件</div>
    </div>
    
    <el-dialog v-model="publishDialog.visible" title="发布到圈子" width="30%">
      <p>
        将 <strong>"{{ publishDialog.nodeTitle }}"</strong>
        及其所有子内容发布到:
      </p>
      <el-select 
        v-model="publishDialog.targetDomainId" 
        placeholder="请选择一个你管理的圈子" 
        style="width: 100%;"
      >
        <el-option
          v-for="domain in myOwnedDomains"
          :key="domain.id"
          :label="domain.name"
          :value="domain.id"
        />
      </el-select>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="publishDialog.visible = false">取消</el-button>
          <el-button type="primary" @click="confirmPublish" :loading="formLoading">
            确认发布
          </el-button>
        </span>
      </template>
    </el-dialog>
  </div>
  
</template>

<script setup>
// --- 您的 Script 逻辑保持完全不变 ---
import { ref, reactive, onMounted, onUnmounted, computed, nextTick } from 'vue';
import apiClient from '@/api';
import { ElMessageBox, ElMessage } from 'element-plus';
import { useAuthStore } from '@/stores/auth';
import RecordingControlBar from '@/components/RecordingControlBar.vue';

import { 
  Folder, Document, FolderAdd, DocumentAdd, Search, 
  ArrowLeft, ArrowRight, MoreFilled ,Loading
} from '@element-plus/icons-vue';
// --- 录音控制相关状态和逻辑 ---

// 1. 核心状态变量
const isRecording = ref(false);      // 是否正在录音
const audioBlob = ref(null);         // 存储录音结果的 Blob 对象

// 2. MediaRecorder 实例和数据块
let mediaRecorder = null;
let audioChunks = [];

// 3. 开始录音 (响应 @start 事件)
const startRecording = async () => {
  try {
    const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
    mediaRecorder = new MediaRecorder(stream);

    mediaRecorder.ondataavailable = (event) => {
      audioChunks.push(event.data);
    };

    mediaRecorder.onstop = () => {
      // 当停止时，将收集到的数据块整合成一个 Blob
      const blob = new Blob(audioChunks, { type: 'audio/webm;codecs=opus' });
      audioBlob.value = blob;
      audioChunks = []; // 清空数据块以便下次录音
      // 关闭麦克风轨道，释放资源
      stream.getTracks().forEach(track => track.stop());
    };

    // 清空旧数据并开始录音
    audioChunks = [];
    mediaRecorder.start();
    isRecording.value = true;
    audioBlob.value = null; // 开始新录音时清空旧的

  } catch (error) {
    console.error("无法获取麦克风权限:", error);
    ElMessage.error("无法启动录音，请检查麦克风权限。");
  }
};

// 4. 停止录音 (响应 @stop 事件)
const stopRecording = () => {
  if (mediaRecorder && mediaRecorder.state === 'recording') {
    mediaRecorder.stop();
    isRecording.value = false;
  }
};

// 5. 重置/重录 (响应 @reset 事件)
const resetRecording = () => {
  audioBlob.value = null;
  isRecording.value = false;
};

// 6. 上传录音 (响应 @upload 事件，这个你已经有了，但要确保它能拿到 audioBlob)
const handleUploadRequest = async () => {
  // 注意：你的子组件没有把 blob 数据传上来，我们改一下
  // 这里我们直接从父组件的 state 中获取
  if (!audioBlob.value) {
    ElMessage.warning('没有可以上传的录音文件。');
    return;
  }
  
  if (!selectedNode.value) return;

  isUploading.value = true;
  const formData = new FormData();
  // 使用父组件的 audioBlob.value
  formData.append('audio_file', audioBlob.value, 'recording.webm');
  formData.append('node_id', selectedNode.value.id);
  
  try {
    await apiClient.post('/recordings/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
    ElMessage.success('上传成功，正在后台处理...');
    resetRecording(); // 上传成功后重置录音条
    fetchRecordingsForCurrentNode(); // 刷新列表
  } catch (error) {
    ElMessage.error('上传失败！');
  } finally {
    isUploading.value = false;
  }
};
// --- AI 助手相关状态 ---
const chatMessages = ref([
    { role: 'assistant', content: '你好！有什么可以帮助你的吗？' }
]);
const userInput = ref('');
const isAiThinking = ref(false);
const chatScrollbarRef = ref(null);
// --- 布局状态管理 (无变化) ---
const isLeftPanelCollapsed = ref(false);
const isRightPanelCollapsed = ref(true);

const toggleLeftPanel = () => {
  isLeftPanelCollapsed.value = !isLeftPanelCollapsed.value;
};

const toggleRightPanel = () => {
  isRightPanelCollapsed.value = !isRightPanelCollapsed.value;
};

// --- 权限管理 (无变化) ---
const canManageContent = computed(() => true); 

// --- 所有已有的业务逻辑和状态 (无变化) ---
const authStore = useAuthStore();
const treeRef = ref(null);
const treeLoading = ref(false);
const selectedNode = ref(null);
const nodeRecordings = ref([]);
const treeProps = {
  label: 'title',
  isLeaf: (data) => data.node_type === 'text',
};
const contextMenu = reactive({
  visible: false,
  style: { top: '0px', left: '0px' },
  node: null,
});
const searchQuery = ref('');
const isSearching = ref(false);
const searchResults = ref([]);
const saveStatus = ref('');
const isUploading = ref(false);
const myOwnedDomains = ref([]);
const formLoading = ref(false);
const publishDialog = reactive({
  visible: false,
  nodeTitle: '',
  sourceNodeId: null,
  targetDomainId: null
});


// 滚动到底部函数
const scrollToBottom = () => {
    nextTick(() => {
        chatScrollbarRef.value?.wrapRef.scrollTo({
            top: chatScrollbarRef.value.wrapRef.scrollHeight,
            behavior: 'smooth'
        });
    });
};
// 发送消息函数
const sendMessage = async () => {
    const prompt = userInput.value.trim();
    if (!prompt || isAiThinking.value) return;

    // 1. 将用户消息添加到聊天记录
    chatMessages.value.push({ role: 'user', content: prompt });
    userInput.value = '';
    isAiThinking.value = true;
    scrollToBottom();

    try {
        // 2. 调用后端 API
        const response = await apiClient.post('/ai/chat', { prompt });
        const aiReply = response.data.reply;

        // 3. 将 AI 回复添加到聊天记录
        chatMessages.value.push({ role: 'assistant', content: aiReply });
    } catch (error) {
        console.error('AI chat failed:', error);
        chatMessages.value.push({
            role: 'assistant',
            content: '抱歉，我遇到了一些问题，请稍后再试。'
        });
        ElMessage.error('与 AI 助手通信失败');
    } finally {
        isAiThinking.value = false;
        scrollToBottom();
    }
};



const getNodeData = (node) => node?.data ? node.data : node;
const debounce = (func, delay = 1500) => {
  let timeout;
  return function(...args) {
    clearTimeout(timeout);
    timeout = setTimeout(() => {
      func.apply(this, args);
    }, delay);
  };
};

const loadNode = async (node, resolve) => {
  const parentId = node.level === 0 ? null : node.data.id;
  try {
    const response = await apiClient.get('/nodes', { params: { parent_id: parentId } });
    resolve(response.data);
  } catch (error) {
    ElMessage.error('加载节点失败');
    resolve([]);
  }
};

const handleNodeClick = async (data) => {
  selectedNode.value = data;
  if (data.node_type === 'text') {
    try {
      await fetchRecordingsForCurrentNode(); 
    } catch (error) {
      console.error("Failed to fetch node recordings", error);
      nodeRecordings.value = [];
    }
  }
};

const handleNodeContextMenu = (event, data, node) => {
  event.preventDefault();
  contextMenu.style.left = event.clientX + 'px';
  contextMenu.style.top = event.clientY + 'px';
  contextMenu.node = node;
  contextMenu.visible = true;
};

const allowDrop = (draggingNode, dropNode, type) => {
  if (dropNode.data.node_type !== 'folder') return false;
  if (type !== 'inner') return false;
  return true;
};

const handleNodeDrop = async (draggingNode, dropNode) => {
  const draggingNodeId = draggingNode.data.id;
  const newParentId = dropNode.data.id;
  try {
    await apiClient.put(`/nodes/${draggingNodeId}/move`, { new_parent_id: newParentId });
    ElMessage.success('移动成功！');
  } catch (error) {
    ElMessage.error('移动失败，将刷新页面以恢复。');
    setTimeout(() => window.location.reload(), 1500);
  }
};

const handleCreateNode = (type, contextNode = null) => {
  ElMessageBox.prompt(`请输入新的${type === 'folder' ? '文件夹' : '文件'}名称`, '新建', {
    inputValidator: (val) => val && val.trim() !== '',
    inputErrorMessage: '名称不能为空',
  }).then(async ({ value }) => {
    const parentData = getNodeData(contextNode);
    let parentId = null;
    if (parentData && parentData.node_type === 'folder') {
      parentId = parentData.id;
    }
    try {
      const response = await apiClient.post('/nodes', {
        parent_id: parentId,
        node_type: type,
        title: value,
      });
      const newNodeData = response.data;
      const parentNodeInTree = parentId ? treeRef.value?.getNode(parentId) : null;
      treeRef.value?.append(newNodeData, parentNodeInTree);
      ElMessage.success('创建成功！');
    } catch (error) {
      ElMessage.error('创建失败');
    }
  }).catch(() => { ElMessage.info('操作已取消'); });
};

const handleRenameNode = (node) => {
  const nodeData = getNodeData(node);
  if (!nodeData) return;
  ElMessageBox.prompt('请输入新名称', '重命名', {
    inputValue: nodeData.title,
  }).then(async ({ value }) => {
    if (!value || value.trim() === '' || value === nodeData.title) return;
    try {
      await apiClient.put(`/nodes/${nodeData.id}`, { title: value });
      const nodeInTree = treeRef.value.getNode(nodeData.id);
      if (nodeInTree) nodeInTree.data.title = value;
      if (selectedNode.value?.id === nodeData.id) selectedNode.value.title = value;
      ElMessage.success('重命名成功');
    } catch (error) { ElMessage.error('重命名失败'); }
  }).catch(() => {});
};

const handleDeleteNode = (node) => {
  const nodeData = getNodeData(node);
  if (!nodeData) return;
  ElMessageBox.confirm(`确定要删除 "${nodeData.title}" 吗？此操作不可逆！`, '警告', {
    type: 'warning',
  }).then(async () => {
    try {
      await apiClient.delete(`/nodes/${nodeData.id}`);
      treeRef.value.remove(nodeData.id);
      if (selectedNode.value?.id === nodeData.id) selectedNode.value = null;
      ElMessage.success('删除成功');
    } catch (error) { ElMessage.error('删除失败'); }
  }).catch(() => {});
};

const saveContent = async () => {
  if (!selectedNode.value) return;
  saveStatus.value = '正在保存...';
  try {
    await apiClient.put(`/nodes/${selectedNode.value.id}`, { content: selectedNode.value.content });
    saveStatus.value = '已保存';
  } catch (error) { 
    saveStatus.value = '保存失败!';
    ElMessage.error('保存失败'); 
  } finally {
    setTimeout(() => { saveStatus.value = ''; }, 2000);
  }
};

const debouncedSave = debounce(saveContent, 5000);

const onContentChange = () => {
  saveStatus.value = '内容已修改...';
  debouncedSave();
};

const performSearch = async () => {
  if (!searchQuery.value.trim()) return;
  isSearching.value = true;
  treeLoading.value = true;
  try {
    const response = await apiClient.get('/nodes/search', { params: { q: searchQuery.value } });
    searchResults.value = response.data;
  } catch (error) { ElMessage.error('搜索失败'); } finally { treeLoading.value = false; }
};

const cancelSearch = () => {
  isSearching.value = false;
  searchResults.value = [];
};



const handleRenameRecording = async (recording) => {
    ElMessageBox.prompt('请输入录音的新名称', '重命名录音', {
        inputValue: recording.title || '',
    }).then(async ({ value }) => {
        if (!value) return;
        try {
            const response = await apiClient.put(`/recordings/${recording.id}`, { title: value });
            const index = nodeRecordings.value.findIndex(r => r.id === recording.id);
            if (index !== -1) nodeRecordings.value[index].title = response.data.title;
            ElMessage.success('重命名成功！');
        } catch (error) { ElMessage.error('重命名失败！'); }
    }).catch(() => {});
};

const handleDeleteRecording = async (recording) => {
    await ElMessageBox.confirm('确定要删除这条录音吗？', '确认删除', { type: 'warning' });
    try {
        await apiClient.delete(`/recordings/${recording.id}`);
        nodeRecordings.value = nodeRecordings.value.filter(r => r.id !== recording.id);
        ElMessage.success('删除成功！');
    } catch (error) { ElMessage.error('删除失败！'); }
};

const isOwnerOfAnyDomain = computed(() => myOwnedDomains.value.length > 0);
const fetchMyOwnedDomains = async () => {
  try {
    const response = await apiClient.get('/domains/my');
    myOwnedDomains.value = response.data.filter(d => d.owner_id === authStore.userId);
  } catch (error) { console.error('获取拥有的圈子失败', error); }
};

const openPublishDialog = () => {
  if (selectedNode.value) {
    publishDialog.visible = true;
    publishDialog.nodeTitle = selectedNode.value.title;
    publishDialog.sourceNodeId = selectedNode.value.id;
    publishDialog.targetDomainId = null;
  }
};

const openPublishDialogFromContextMenu = async () => {
  const nodeToPublish = contextMenu.node;
  contextMenu.visible = false;
  if (!nodeToPublish) return;
  if (myOwnedDomains.value.length === 0) await fetchMyOwnedDomains();
  publishDialog.visible = true;
  publishDialog.nodeTitle = nodeToPublish.data.title;
  publishDialog.sourceNodeId = nodeToPublish.data.id;
  publishDialog.targetDomainId = null;
};

const confirmPublish = async () => {
  if (!publishDialog.targetDomainId) {
    ElMessage.warning('请选择一个目标圈子');
    return;
  }
  formLoading.value = true;
  try {
    await apiClient.post(`/domains/${publishDialog.targetDomainId}/publish`, {
      source_node_id: publishDialog.sourceNodeId,
    });
    ElMessage.success('发布成功！内容已复制到圈子中。');
    publishDialog.visible = false;
  } catch (error) {
    ElMessage.error(error.response?.data?.error || '发布失败');
  } finally {
    formLoading.value = false;
  }
};

const closeContextMenu = () => { contextMenu.visible = false; };
onMounted(() => {
  document.addEventListener('click', closeContextMenu);
  fetchMyOwnedDomains();
});
onUnmounted(() => {
  document.removeEventListener('click', closeContextMenu);
  clearAllPolls();
});

const pollingTimers = new Map();
const MAX_POLLING_ATTEMPTS = 10;

// 【核心修改】
const fetchRecordingsForCurrentNode = async () => {
    if (!selectedNode.value || selectedNode.value.node_type !== 'text') {
        clearAllPolls();
        return;
    }

    try {
        const response = await apiClient.get(`/nodes/${selectedNode.value.id}/recordings`);
        const newRecordings = response.data;
        nodeRecordings.value = newRecordings; // 更新UI

        const activePollIds = new Set();

        newRecordings.forEach(rec => {
            let needsPolling = false;
            
            // 【修改】将两种状态的监控分开
            
            // --- 1. 监控主任务 (文件上传) ---
            if (rec.status === 'processing') {
                needsPolling = true; // 只要主任务还在处理，就需要轮询
                const timerInfo = pollingTimers.get(rec.id);
                if (timerInfo) {
                    timerInfo.uploadAttempts++; // 使用独立的计数器
                    if (timerInfo.uploadAttempts > MAX_POLLING_ATTEMPTS) {
                        // 【关键】上传任务超时是严重问题
                        clearInterval(timerInfo.intervalId);
                        pollingTimers.delete(rec.id);
                        rec.status = 'failed';
                        ElMessage.error(`录音 ${rec.title || rec.id} 上传处理超时，请删除后重试。`);
                    }
                }
            }

            // --- 2. 监控 AI 任务 ---
            // 只有在主任务完成后，才开始监控 AI 任务
            if (rec.status === 'completed' && (rec.ai_status === 'pending' || rec.ai_status === 'processing')) {
                needsPolling = true; // AI 任务也需要轮询
                const timerInfo = pollingTimers.get(rec.id);
                if (timerInfo) {
                    timerInfo.aiAttempts++; // 使用独立的 AI 计数器
                    if (timerInfo.aiAttempts > MAX_POLLING_ATTEMPTS * 2) { // AI 超时可以设置更长，例如 1 分钟
                        // 【关键】AI 任务超时不是严重问题
                        clearInterval(timerInfo.intervalId);
                        pollingTimers.delete(rec.id);
                        rec.ai_status = 'failed';
                        // 【修改】使用警告提示，而不是错误
                        ElMessage.warning(`录音 ${rec.title || rec.id} 的AI识别超时，但录音已保存，您可稍后重试识别。`);
                    }
                }
            }
            
            // --- 总控逻辑 ---
            if (needsPolling) {
                activePollIds.add(rec.id);
                if (!pollingTimers.has(rec.id)) {
                    startPolling(rec.id); // 如果需要轮询且没有定时器，则启动
                }
            }
        });

        // 清理逻辑保持不变
        for (const id of pollingTimers.keys()) {
            if (!activePollIds.has(id)) {
                clearInterval(pollingTimers.get(id).intervalId);
                pollingTimers.delete(id);
            }
        }

    } catch (error) {
        console.error("Failed to fetch node recordings", error);
    }
}

// startPolling 函数需要修改，以初始化两个计数器
const startPolling = (recordingId) => {
    if (pollingTimers.has(recordingId)) return;
    
    const intervalId = setInterval(fetchRecordingsForCurrentNode, 3000);
    // 【修改】初始化两个独立的计数器
    pollingTimers.set(recordingId, { 
        intervalId, 
        uploadAttempts: 0, 
        aiAttempts: 0 
    });
}

// 新增一个清理所有定时器的函数
const clearAllPolls = () => {
    for (const timerInfo of pollingTimers.values()) {
        clearInterval(timerInfo.intervalId);
    }
    pollingTimers.clear();
}

// `retryTranscribe` 函数现在只负责发送请求和更新初始状态
const retryTranscribe = async (recording) => {
    if (recording.ai_status === 'pending' || recording.ai_status === 'processing') return;

    try {
        recording.ai_status = 'pending';
        await apiClient.post(`/recordings/${recording.id}/transcribe`);
        ElMessage.success('已成功加入识别队列！');
        
        // 立即启动一次轮询来更新状态和管理定时器
        fetchRecordingsForCurrentNode();

    } catch (error) {
        ElMessage.error('操作失败，请重试');
        recording.ai_status = 'failed'; 
    }
}


</script>

<style scoped>
/* ==========================================================================
   核心布局修复 (基于 Flexbox)
   ========================================================================== */

/* 1. 让中间面板成为一个 Flex 容器，以便其子元素可以撑满高度 */
.center-panel {
  display: flex;
  flex-direction: column;
  overflow: hidden; /* 防止子元素溢出 */
  /* 保留您原有的卡片样式 */
  flex: 1 1 auto;
  background-color: #fff;
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0,0,0,0.08);
}

/* 2. 让内容包裹器撑满中间面板 */
.content-view-wrapper { 
  flex: 1 1 auto; 
  min-height: 0; 
  display: flex; 
  flex-direction: column; 
}

/* 3. 【解决问题一】让文本编辑区自动伸展 */
.editor-area {
  flex-grow: 1; /* 核心：占据所有剩余的垂直空间 */
  min-height: 200px; /* 保证一个最小的可视高度 */
  padding: 16px;
  display: flex; /* 让内部的 el-input 也撑满 */
}

/* 穿透 el-input 组件样式，让 textarea 元素真正撑满其父容器(.editor-area) */
:deep(.editor-area .el-textarea),
:deep(.editor-area .el-textarea__inner) {
  height: 100%;
  border: none;
  box-shadow: none;
  padding: 0;
  resize: none;
  background-color: transparent;
  font-size: 16px;
  line-height: 1.7;
}


/* 4. 【解决问题二】让录音区高度固定且内部可滚动 */
.recording-section {
  /* 1. 让整个录音区成为一个垂直方向的 Flex 容器 */
  display: flex;
  flex-direction: column;
  
  /* 2. 高度保持不变，让子元素来分配这个高度 */
  height: 400px; /* 您可以根据需要调整这个总高度 */
  border-top: 1px solid #eee;
}


.recording-section .recording-controls {
  padding: 16px;
  flex-shrink: 0; /* 控件区域高度固定 */
}

.recording-section h3 {
  margin: 0 0 16px 0;
  font-size: 16px;
  padding: 0 16px;
  flex-shrink: 0;
}

.history-list { flex: 1 1 auto; min-height: 0; padding: 0 16px 16px; }
:deep(.el-table) { height: 100%; }
:deep(.el-table__body-wrapper) { max-height: 100%; overflow: auto; }
.recording-actions {
  display: flex;
  align-items: center; /* 垂直居中 */
  justify-content: space-between; /* 播放器和按钮组两端对齐 */
  width: 100%;
}
.action-buttons {
  display: flex;
  align-items: center;
  flex-shrink: 0; /* 防止按钮被压缩 */
  margin-left: 10px;
}
/* 1280 以下：默认收起右侧 AI 面板；1100 以下：收起左侧面板 */
@media (max-width: 1280px) {
  .content-layout-container.with-right { padding-right: 16px; } /* 右侧展开时也不挤压中间 */
}
@media (max-width: 1100px) {
  .left-panel { flex-basis: 0; opacity: 0; }
  .left-panel-toggle { left: 0; }
}

/* ==========================================================================
   保留您其他的样式 (头部、按钮组、右键菜单等)
   ========================================================================== */
.my-content-page, .content-layout-container, .left-panel, .right-panel,
.panel-header, .panel-content, .panel-search, .panel-actions, .panel-tree,
.tree-scrollbar, .custom-tree, :deep(.custom-tree .el-tree-node__content),
:deep(.custom-tree .el-tree-node.is-current > .el-tree-node__content),
.custom-tree-node .el-icon, .left-panel-toggle, .center-panel-header,
.filename, .actions-group, .save-status,  .context-menu,
.menu-item, .menu-separator, .search-results, .search-result-item,
.search-empty, .ai-assistant, .chat-history, .chat-message, .message-bubble,
.loading-bubble, .chat-input, .recording-actions, .left-panel.collapsed, 
.left-panel-toggle.collapsed
 {
  /* 这里粘贴您提供的所有其他样式，确保它们保持不变 */
  /* 为节省篇幅，此处省略，请将您原有的、与核心布局无关的样式代码复制到这里 */
}
.right-panel.collapsed {
  transform: translateX(100%);
  opacity: 0;
  pointer-events: none;
}
/* 中间容器在右侧展开时预留内边距，避免被 fixed 面板遮挡 */
.content-layout-container.with-right {
  padding-right: calc(var(--right-panel-w) + 16px);
}
.right-panel-toggle {
  position: fixed;
  top: calc(var(--header-h) + 8px);
  right: 24px;
  z-index: 100;
}
/* --- 您可以从这里开始，将您原有样式中除了上面已修正的部分，全部粘贴过来 --- */
/*
 *  【全新修正的样式】
 *  这部分代码实现了您描述的沉浸式布局
 */

.my-content-page {
  /* 使用 Flexbox 布局 */
  display: flex;
  flex-direction: column; /* 垂直排列子元素 */
  
  /* 占据所有可用空间 */
  width: 100%;
  height: 100%; /* 这个 height: 100% 很关键 */
  
  position: relative; /* 为内部的 fixed/absolute 定位提供上下文 */
  /*overflow: hidden;  隐藏内部可能产生的滚动条 */
}

.content-layout-container {
  flex: 1;
  display: flex;
  height: 100%;
  overflow: hidden;
  padding: 16px;
  padding-top: 0;
  gap: 16px;
  max-width: 1580px; 
  position: relative;
}

/* 面板通用样式 */
.left-panel {
  flex: 0 0 var(--left-panel-w);
  background: transparent;
  overflow: hidden;
  transition: flex-basis .25s ease, opacity .25s ease;
}
.left-panel, .right-panel {
  display: flex;
  flex-direction: column;
  transition: all 0.3s ease-in-out;
  overflow: hidden;
}
.panel-header {
  flex-shrink: 0; 
  padding: 16px;
  padding-bottom: 8px;
}
.panel-header h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: #333;
}
.panel-content {
  flex-grow: 1;
  min-height: 0; 
  display: flex;
  flex-direction: column;
}

/* 左侧面板完全融入背景 */
.left-panel {
  flex: 0 0 300px;
  background-color: transparent; 
}
.left-panel.collapsed {
  flex-basis: 0;
  min-width: 0;
  opacity: 0;
}
.left-panel .panel-header, .left-panel .panel-content {
  background-color: transparent; 
  border: none;
  padding-left: 0;
  padding-right: 0;
}
.left-panel .panel-search, .left-panel .panel-actions { flex-shrink: 0; padding: 8px; }
.left-panel .panel-actions .el-button { width: 100%; margin: 4px 0 !important; }
.left-panel .panel-tree { flex-grow: 1; }
.tree-scrollbar { height: 100%; }
.custom-tree, :deep(.custom-tree .el-tree-node__content) {
  background-color: transparent;
}
:deep(.custom-tree .el-tree-node.is-current > .el-tree-node__content) {
  background-color: rgba(64, 158, 255, 0.1); 
}
.custom-tree-node .el-icon { margin-right: 8px; }


/* 右侧 AI 助手面板也融入背景 */
.right-panel {
  position: fixed;
  right: 0;
  flex: 0 0 350px;
  height: calc(100vh - 76px);
  width: 410px;
  background-color: transparent;
}
.right-panel.collapsed { 
  flex-basis: 0;
  min-width: 0;
  padding: 0;
  margin-left: -16px; 
  opacity: 0;
}
.right-panel .panel-header, .right-panel .panel-content {
  background-color: transparent;
  border: none;
}


/* 左侧折叠按钮基于变量定位 */
.left-panel-toggle {
  position: absolute;
  top: 50%;
  left: calc(var(--left-panel-w) - 16px); /* 与面板边缘贴边，16px为按钮宽度的一半微调 */
  transform: translateY(-50%);
  width: 24px;
  height: 48px;
  background: #f0f2f5;
  border: 1px solid #dcdfe6;
  border-left: none;
  border-top-right-radius: 6px;
  border-bottom-right-radius: 6px;
  display: flex; align-items: center; justify-content: center;
  cursor: pointer; z-index: 10;
  transition: left .25s ease, background-color .2s ease;
  box-shadow: 2px 0 5px rgba(0,0,0,0.04);
}
.left-panel-toggle:hover {
  background-color: #e9e9eb;
}
.left-panel-toggle.collapsed {
    left: 0px; 
    border-left: 1px solid #dcdfe6;
}

/* 中间面板头部 */
.center-panel-header { display: flex; justify-content: space-between; align-items: center; padding: 16px; border-bottom: 1px solid #eee; flex-shrink: 0; }
.filename { margin: 0; font-size: 20px; font-weight: 600; display: flex; align-items: center; }
.filename .el-icon { margin-right: 8px; }
.actions-group { display: flex; align-items: center; gap: 10px; }


/* 右侧面板开关按钮 */
.right-panel {
  position: fixed;
  top: var(--header-h);
  right: 0;
  width: var(--right-panel-w);
  height: calc(100vh - var(--header-h));
  background-color: transparent;
  display: flex;
  flex-direction: column;
  transition: opacity .25s ease, transform .25s ease;
  will-change: transform, opacity;
}

/* 其他样式 */
.context-menu { position: fixed; background: white; border: 1px solid #ccc; box-shadow: 0 2px 10px rgba(0,0,0,0.1); border-radius: 4px; z-index: 1000; padding: 5px 0; }
.menu-item { font-size: 14px; padding: 8px 15px; cursor: pointer; }
.menu-item:hover { background-color: #ecf5ff; color: #409eff; }
.menu-separator { border-top: 1px solid #eee; margin: 5px 0; }
.search-results { padding: 10px; }
.search-result-item { padding: 8px 12px; cursor: pointer; display: flex; align-items: center; border-radius: 4px; }
.search-result-item:hover { background-color: rgba(0,0,0,0.04); }
.search-result-item .el-icon { margin-right: 8px; }
.search-empty { color: #909399; text-align: center; padding: 20px; }

/* AI 助手特定样式 */
.ai-assistant {
  display: flex;
  flex-direction: column;
  height: 100%;
}
.chat-history {
  flex-grow: 1;
  padding: 10px;
  overflow-y: auto;
}
.chat-message {
  display: flex;
  margin-bottom: 15px;
}
.chat-message.user {
  justify-content: flex-end;
}
.chat-message.assistant {
  justify-content: flex-start;
}
.message-bubble {
  max-width: 80%;
  padding: 10px 15px;
  border-radius: 18px;
  line-height: 1.5;
}
.chat-message.user .message-bubble {
  background-color: #409eff;
  color: white;
  border-bottom-right-radius: 4px;
}
.chat-message.assistant .message-bubble {
  background-color: #f0f2f5;
  color: #333;
  border-bottom-left-radius: 4px;
}
.loading-bubble {
    animation: blink 1.5s infinite;
}
@keyframes blink {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.5; }
}

.chat-input {
  flex-shrink: 0;
  padding: 10px;
  border-top: 1px solid #e4e7ed;
}

.recognition-text-popover p {
  margin: 0;
  padding: 5px;
  line-height: 1.7;
  font-size: 14px;
  color: #333;
  white-space: pre-wrap; /* 保持文本换行 */
  max-height: 300px; /* 防止文本过长 */
  overflow-y: auto;
}

.status-indicator {
    display: flex;
    align-items: center;
    justify-content: center;
    color: #909399;
    font-size: 12px;
}
.status-indicator .el-icon {
    margin-right: 4px;
}

</style>
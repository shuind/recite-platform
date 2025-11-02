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
          <!-- 头部：把可编辑标题放到 header 的左侧 -->
          <div class="center-panel-header">
            <!-- 用 el-input 直接作为标题编辑 -->
            <el-input
              v-if="selectedNode && selectedNode.node_type === 'text'"
              class="title-input"
              v-model="selectedNode.title"
              placeholder="输入标题"
            />
            <div class="actions-group" v-if="canManageContent">
              <span class="save-status">{{ saveStatus }}</span>
              <el-button @click="saveContent">保存内容</el-button>
              <el-button @click="handleRenameNode(selectedNode)">重命名</el-button>
              <el-button type="danger" @click="handleDeleteNode(selectedNode)">删除</el-button>
              <el-button type="primary" @click="openPublishDialog" v-if="isOwnerOfAnyDomain">发布到圈子...</el-button>
              <el-button @click="triggerUploadImage">上传图片</el-button>
              <el-button @click="openExportDialog">导出</el-button>
            </div>
          </div>

          <!-- 【UI优化】沉浸式编辑区 -->
          <div class="editor-area">
              <!-- 【修改】移除 el-tabs -->
              <!-- 直接放置 v-md-editor，并添加图片上传的处理器 -->
              <v-md-editor
                ref="mdRef"
                v-model="selectedNode.content"
                class="md-editor"
                @change="onContentChange"
                @upload-image="handleImageUpload"  
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

    <el-dialog v-model="exportDialog.visible" title="导出内容" width="30%" destroy-on-close>
    <p>
      将从 <strong>"{{ selectedNode?.title }}"</strong> 开始，递归导出所有子内容。
    </p>
    <el-form>
      <el-form-item label="导出格式">
        <el-select v-model="exportDialog.format" placeholder="请选择格式">
          <el-option label="Markdown (.md)" value="md" />
          <el-option label="纯文本 (.txt)" value="txt" />
          <!-- PDF 导出通常需要后端支持强大库，这里先作为选项 -->
          <el-option label="PDF (.pdf)" value="pdf" />
        </el-select>
      </el-form-item>
    </el-form>
    <template #footer>
      <span class="dialog-footer">
        <el-button @click="exportDialog.visible = false">取消</el-button>
        <el-button type="primary" @click="confirmExport" :loading="exportDialog.loading">
          {{ exportDialog.loading ? '正在处理中...' : '开始导出' }}
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

const assetUrls = computed({
  get: () => Array.isArray(selectedNode.value?.asset_urls) ? selectedNode.value.asset_urls : [],
  set: (v) => { selectedNode.value.asset_urls = v; onContentChange(); }
});

/** @param {{ url: string }} res */   // <<< 这是 JSDoc，可要可不要
const onAssetUploaded = (res) => {
  if (res && res.url) {
    assetUrls.value = [...assetUrls.value, res.url]
  }
}

// 保存逻辑：把 content_type/code_lang/asset_urls 一起提交
const saveContent = async () => {
  if (!selectedNode.value) return;
  saveStatus.value = '正在保存...';
  try {
    await apiClient.put(`/nodes/${selectedNode.value.id}`, {
      content: selectedNode.value.content,
      content_type: selectedNode.value.content_type,
      code_lang: selectedNode.value.code_lang,
      asset_urls: assetUrls.value,
    });
    saveStatus.value = '已保存';
  } catch {
    saveStatus.value = '保存失败!';
  } finally {
    setTimeout(() => saveStatus.value = '', 1500);
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

// 【新增】导出功能的状态管理
const exportDialog = reactive({
  visible: false,
  format: 'md',    // 默认格式
  loading: false,  // 是否正在导出中
  jobId: null,     // 后端返回的任务ID
});
// 1. 打开导出对话框
const openExportDialog = () => {
  if (!selectedNode.value) {
    ElMessage.warning('请先从左侧选择一个要导出的文件或文件夹。');
    return;
  }
  exportDialog.visible = true;
  exportDialog.loading = false;
  exportDialog.format = 'md'; // 重置
};
// 2. 确认导出，向后端发起请求
const confirmExport = async () => {
  if (!selectedNode.value) return;

  exportDialog.loading = true;
  try {
    // 假设后端接口为 POST /exports
    const response = await apiClient.post('/exports', {
      root_node_id: selectedNode.value.id, // 告诉后端从哪个节点开始
      format: exportDialog.format,         // 告诉后端导出的格式
    });
    
    // 后端应返回一个任务 ID，用于后续查询状态
    const jobId = response.data.job_id;
    if (jobId) {
        ElMessage.info('导出任务已创建，正在后台处理...');
        exportDialog.visible = false;
        pollExportStatus(jobId); // 开始轮询任务状态
    } else {
        throw new Error("后端未返回任务ID");
    }

  } catch (error) {
    console.error('发起导出失败:', error);
    ElMessage.error(error.response?.data?.error || '发起导出失败，请稍后重试。');
    exportDialog.loading = false;
  }
};

// 3. 轮询导出任务状态
const pollExportStatus = (jobId) => {
  const timer = setInterval(async () => {
    try {
      // 假设后端状态查询接口为 GET /exports/{jobId}
      const response = await apiClient.get(`/exports/${jobId}`);
      const status = response.data.status;

      if (status === 'completed') {
        clearInterval(timer);
        ElMessage.success('导出成功！即将开始下载...');
        // 后端应提供一个下载链接
        window.location.href = response.data.download_url; 
        // 或者直接是 /api/exports/{jobId}/download
      } else if (status === 'failed') {
        clearInterval(timer);
        ElMessage.error(response.data.error_message || '导出任务失败！');
      }
      // 如果 status 是 'processing' 或 'pending'，则什么都不做，继续等待下一次轮询

    } catch (error) {
      clearInterval(timer);
      console.error('轮询导出状态失败:', error);
      ElMessage.error('无法获取导出状态，请检查网络或联系管理员。');
    }
  }, 3000); // 每3秒查询一次
};

// 【新增】处理 v-md-editor 的图片上传事件
const handleImageUpload = async (event, insertImage, files) => {
  // `files` 是一个 File 对象数组，我们处理第一个
  if (!files.length) return;
  const file = files[0];

  // 1. 创建 FormData 用于上传
  const formData = new FormData();
  formData.append('file', file);

  try {
    // 2. 调用您已有的统一资源上传接口
    const response = await apiClient.post('/assets/upload', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });

    const imageUrl = response.data.url;
    if (!imageUrl) {
      throw new Error('Upload succeeded but no URL was returned.');
    }

    // 3. 【关键】使用 v-md-editor 提供的回调函数插入图片
    // 这会在编辑器的光标位置插入 ![file.name](imageUrl)
    insertImage({
      url: imageUrl,
      desc: file.name, // 图片的 alt 文本，默认为文件名
      // width: 'auto',
      // height: 'auto',
    });
    ElMessage.success('图片上传成功！');

  } catch (error) {
    console.error('图片上传失败:', error);
    ElMessage.error('图片上传失败，请检查网络或联系管理员。');
  }
};

const mdRef = ref(null);

const insertAtCursor = (md) => {
  const textarea =
    mdRef.value?.$el?.querySelector?.('.v-md-editor textarea') ||
    document.querySelector('.v-md-editor textarea');
  if (!textarea) {
    selectedNode.value.content = (selectedNode.value.content || '') + '\n' + md + '\n';
    return;
  }
  const start = textarea.selectionStart || 0;
  const end = textarea.selectionEnd || start;
  const val = selectedNode.value.content || '';
  selectedNode.value.content = val.slice(0, start) + md + val.slice(end);
  nextTick(() => {
    const pos = start + md.length;
    textarea.selectionStart = textarea.selectionEnd = pos;
    textarea.focus();
  });
};

const uploadImageFile = async (file) => {
  const formData = new FormData();
  formData.append('file', file);
  const res = await apiClient.post('/assets/upload', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  });
  const url = res.data && res.data.url;
  if (!url) throw new Error('no url');
  return url;
};

const triggerUploadImage = () => {
  const input = document.createElement('input');
  input.type = 'file';
  input.accept = 'image/*';
  input.onchange = async () => {
    const file = input.files && input.files[0];
    if (!file) return;
    try {
      const url = await uploadImageFile(file);
      insertAtCursor(`![${file.name}](${url})`);
      ElMessage.success('已插入图片');
    } catch (e) {
      ElMessage.error('图片上传失败');
    }
  };
  input.click();
};

const onPaste = async (e) => {
  if (!e.clipboardData) return;
  const file = e.clipboardData.files && e.clipboardData.files[0];
  if (file && file.type.startsWith('image/')) {
    e.preventDefault();
    try {
      const url = await uploadImageFile(file);
      insertAtCursor(`![${file.name}](${url})`);
      ElMessage.success('已插入图片');
    } catch (e) { ElMessage.error('图片上传失败'); }
    return;
  }
  const text = e.clipboardData.getData('text');
  if (text && /^https?:\/\/.+\.(png|jpe?g|gif|webp|svg)(\?.*)?$/i.test(text.trim())) {
    e.preventDefault();
    insertAtCursor(`![](${text.trim()})`);
  }
};

const onDrop = async (e) => {
  e.preventDefault();
  const file = e.dataTransfer && e.dataTransfer.files && e.dataTransfer.files[0];
  if (file && file.type.startsWith('image/')) {
    try {
      const url = await uploadImageFile(file);
      insertAtCursor(`![${file.name}](${url})`);
      ElMessage.success('已插入图片');
    } catch (e) { ElMessage.error('图片上传失败'); }
  }
};
const onDragOver = (e) => e.preventDefault();

onMounted(() => {
  nextTick(() => {
    const editor = mdRef.value?.$el || document.querySelector('.v-md-editor');
    if (!editor) return;
    editor.addEventListener('paste', onPaste);
    editor.addEventListener('drop', onDrop);
    editor.addEventListener('dragover', onDragOver);
  });
});
onUnmounted(() => {
  const editor = mdRef.value?.$el || document.querySelector('.v-md-editor');
  if (!editor) return;
  editor.removeEventListener('paste', onPaste);
  editor.removeEventListener('drop', onDrop);
  editor.removeEventListener('dragover', onDragOver);
});

</script>

<style scoped>
/* ==========================================================================
   核心布局 (实现沉浸式三栏)
   ========================================================================== */

/* 页面总容器，使用 Flex 撑满视口 */
.my-content-page {
  display: flex;
  height: 100%;
  padding: 16px;
  box-sizing: border-box;
  background-color: #f0f2f5; /* 整个页面的浅灰色背景，类似 CSDN */
}

/* 布局主容器 */
.content-layout-container {
  flex: 1;
  display: flex;
  gap: 16px; /* 面板之间的间距 */
  position: relative;
  overflow: hidden;
}

/* 左右面板通用样式 (透明背景) */
.left-panel, .right-panel {
  display: flex;
  flex-direction: column;
  background-color: transparent; /* 关键：使其融入页面背景 */
  transition: all 0.3s ease-in-out;
}

/* 左侧面板 */
.left-panel {
  flex: 0 0 300px; /* 固定宽度 */
}
.left-panel.collapsed {
  flex-basis: 0;
  min-width: 0;
  opacity: 0;
  margin-right: -16px; /* 收起时消除 gap 影响 */
}

/* 中间主内容区 (白色卡片) */
.center-panel {
  flex: 1;
  display: flex; /* 让内部元素也能使用 Flex */
  flex-direction: column;
  background-color: #fff; /* 关键：白色卡片背景 */
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0,0,0,0.08); /* 添加阴影提升质感 */
  overflow: hidden; /* 防止内容溢出圆角 */
}

/* 右侧 AI 助手 */
.right-panel {
  flex: 0 0 350px;
}
.right-panel.collapsed {
  flex-basis: 0;
  min-width: 0;
  opacity: 0;
  margin-left: -16px;
}


/* ==========================================================================
   中间编辑区优化 (实现 CSDN 风格的沉浸式编辑)
   ========================================================================== */

/* 内容包裹器，使用 Flex 撑满中间面板 */
.content-view-wrapper {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.editor-area {
  flex: 1 1 auto;     /* 占据剩余空间 */
  min-height: 0;      /* 允许被压缩，防止溢出 */
  display: flex;      /* 关键：用 flex 来分配高度 */
  padding: 24px;
}

/* 让 el-tabs 和其内容也撑满 */
:deep(.editor-area .el-tabs),
:deep(.editor-area .el-tabs__content),
:deep(.editor-area .el-tab-pane) {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}

/* 穿透修改内部组件样式，让它们真正“沉浸” */
:deep(.editor-area .el-textarea),
:deep(.editor-area .el-textarea__inner) {
  flex: 1; /* 关键：让 textarea 自身也撑满 */
  border: none;
  box-shadow: none;
  padding: 0;
  resize: none;
  background-color: transparent;
  font-size: 16px; /* 提升写作体验 */
  line-height: 1.7;
}

/* 让 v-md-editor 用 flex 规则填满父容器，而不是依赖 height:100% */
:deep(.editor-area .v-md-editor) {
  flex: 1 1 0;        /* 高度随父容器分配 */
  min-height: 0;
  height: auto;       /* 避免百分比高度为 0 的问题 */
  width: 100%;
  box-sizing: border-box;
}


:deep(.editor-area .v-md-editor .v-md-editor__editor-wrapper) {
  border-right: 1px solid #e8e8e8;
}


/* ==========================================================================
   其他区域样式 (保持不变或微调)
   ========================================================================== */

.panel-header {
  padding: 16px;
  flex-shrink: 0;
}
.panel-header h3 { margin: 0; }
.panel-content {
  flex-grow: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  padding: 0 16px 16px;
}
.panel-actions { margin-bottom: 10px; }
.panel-actions .el-button { width: 100%; margin: 4px 0 !important; }
.panel-tree { flex: 1; min-height: 0; }
.tree-scrollbar { height: 100%; }
.custom-tree, :deep(.custom-tree .el-tree-node__content) { background-color: transparent; }
:deep(.custom-tree .el-tree-node.is-current > .el-tree-node__content) { background-color: rgba(64, 158, 255, 0.1); }

.center-panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 24px;
  border-bottom: 1px solid #eee;
  flex-shrink: 0;
}
/* 让标题输入框更有设计感 */
.title-input {
  font-size: 24px;
  font-weight: 600;
}
:deep(.title-input .el-input__wrapper) {
  box-shadow: none !important;
  padding: 0;
}
:deep(.title-input .el-input__inner) {
  height: auto;
  line-height: inherit;
  font-family: inherit;
  font-weight: inherit;
  color: #303133;
}
.actions-group { display: flex; align-items: center; gap: 10px; }


/* 录音区 */
.practice-area {
  padding: 16px 24px;
  border-top: 1px solid #eee;
}
.recording-section {
  height: 300px; /* 固定一个高度 */
  display: flex;
  flex-direction: column;
  border-top: 1px solid #eee;
  padding: 16px 24px;
}
.recording-section h3 { margin: 0 0 10px 0; }
.history-list { flex: 1; min-height: 0; }


/* --- 保留你原有的其他样式，如右键菜单、AI助手等 --- */
.left-panel-toggle { position: absolute; top: 50%; left: 300px; transform: translateY(-50%); z-index: 10; cursor: pointer; background: #fff; border: 1px solid #dcdfe6; border-radius: 0 50% 50% 0; width: 24px; height: 48px; display: flex; align-items: center; justify-content: center; box-shadow: 2px 0 5px rgba(0,0,0,0.05); transition: all .3s ease; }
.left-panel-toggle.collapsed { left: 0; border-radius: 0 6px 6px 0; }
.right-panel-toggle { position: fixed; top: 80px; right: 24px; z-index: 1001; }
.context-menu { position: fixed; background: white; border: 1px solid #ccc; box-shadow: 0 2px 10px rgba(0,0,0,0.1); border-radius: 4px; z-index: 1000; padding: 5px 0; }
.menu-item { font-size: 14px; padding: 8px 15px; cursor: pointer; }
.menu-item:hover { background-color: #ecf5ff; color: #409eff; }
.menu-separator { border-top: 1px solid #eee; margin: 5px 0; }
.ai-assistant, .chat-history, .chat-message, .message-bubble, .chat-input { /* 你的样式... */ }

.ai-assistant { display: flex; flex-direction: column; height: 100%; }
.chat-history { flex-grow: 1; padding: 10px; overflow-y: auto; }
.chat-message { display: flex; margin-bottom: 15px; }
.chat-message.user { justify-content: flex-end; }
.chat-message.assistant { justify-content: flex-start; }
.message-bubble { max-width: 80%; padding: 10px 15px; border-radius: 18px; line-height: 1.5; }
.chat-message.user .message-bubble { background-color: #409eff; color: white; border-bottom-right-radius: 4px; }
.chat-message.assistant .message-bubble { background-color: #f0f2f5; color: #333; border-bottom-left-radius: 4px; }
.loading-bubble { animation: blink 1.5s infinite; }
@keyframes blink { 0%, 100% { opacity: 1; } 50% { opacity: 0.5; } }
.chat-input { flex-shrink: 0; padding: 10px; border-top: 1px solid #e4e7ed; }


.save-status {
  display: inline-flex;
  align-items: center;
  height: 28px;            /* 固定高度，避免把 header 顶高 */
  padding: 0 8px;
  border-radius: 4px;
  background: #f5f7fa;
  color: #909399;
  font-size: 12px;
  line-height: 1;          /* 防止文字行高拉伸 */
  white-space: nowrap;     /* 不换行，避免竖排堆字 */
  writing-mode: horizontal-tb;  /* 强制横排，防止继承到竖排 */
  min-width: 72px;         /* 可选：宽度稳定一点 */
}
</style>
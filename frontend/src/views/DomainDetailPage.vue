<template>
  <!-- 布局和 MyContentPage 完全一样 -->
  <div class="my-content-page">
    <div class="content-layout-container">

      <!-- 左侧折叠按钮 -->
      <div 
        class="left-panel-toggle" 
        :class="{ collapsed: isLeftPanelCollapsed }"
        @click="toggleLeftPanel"
      >
        <el-icon><ArrowLeft v-if="!isLeftPanelCollapsed" /><ArrowRight v-else /></el-icon>
      </div>
      
      <!-- 左侧导航面板 -->
      <div class="left-panel" :class="{ collapsed: isLeftPanelCollapsed }">
        <div class="panel-header">
          <h3 v-if="domainInfo">{{ domainInfo.name }}</h3>
        </div>
        <div class="panel-content">
          <div class="panel-search">
            <el-input
              v-model="searchQuery"
              placeholder="搜索圈子内容..."
              :prefix-icon="Search"
              @keyup.enter="performSearch"
              @clear="cancelSearch"
              clearable
            />
          </div>
          <div class="panel-actions" v-if="canManageContent">
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
                :draggable="canManageContent" 
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

      <!-- 中间主内容区 -->
      <div class="center-panel">
        <div v-if="selectedNode && selectedNode.node_type === 'text'" class="content-view-wrapper">
          <div class="center-panel-header">
            <h2 class="filename">{{ selectedNode.title }}</h2>
            <div class="actions-group" v-if="canManageContent">
              <span class="save-status">{{ saveStatus }}</span>
              <el-button @click="saveContent">保存内容</el-button>
              <el-button @click="handleRenameNode(selectedNode)">重命名</el-button>
              <el-button type="danger" @click="handleDeleteNode(selectedNode)">删除</el-button>
            </div>
          </div>
          <div class="editor-area">
            <div v-if="!canManageContent" class="readonly-content">
              {{ selectedNode.content }}
            </div>
            <el-input
              v-else
              v-model="selectedNode.content"
              type="textarea"
              placeholder="在这里输入或粘贴你的朗读材料..."
              @input="onContentChange"
            />
          </div>
          <div class="practice-area"> 
              <div class="recording-controls">
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
            <div class="section-header">
               <!-- 左侧部分：视图切换器 -->
              <div class="left-controls">
                <el-radio-group v-model="tableViewMode" size="small" @change="onViewModeChange">
                  <el-radio-button value="history">我的录音</el-radio-button>
                  <el-radio-button value="featured">本篇精选</el-radio-button>
                </el-radio-group>
              </div>

              <!-- 右侧部分：独立操作按钮 -->
              <div class="right-controls">
                <el-button @click="openCommentDrawer(selectedNode)" :icon="ChatDotRound">
                  评论区 ({{ selectedNode.comments_count || 0 }})
                </el-button>
              </div>
            </div>
            
            <div class="history-list" v-loading="featuredLoading">
              <el-table :data="tableData" stripe height="100%"> 

                <!-- 【动态列】朗读者，只在精选模式下显示 -->
                <el-table-column 
                  v-if="tableViewMode === 'featured'" 
                  label="用户" 
                  width="150"
                >
                  <template #default="scope">
                    <!-- 【修正】scope.row.User -> scope.row.user -->
                    <router-link 
                      :to="{ name: 'profile', params: { userId: scope.row.user.id } }" 
                      class="username-link"
                    >
                      {{ scope.row.user.username }}
                    </router-link>
                  </template>
                </el-table-column>
                
                <el-table-column label="录音标题" prop="title" width="200">
                  <template #default="scope">
                    <span>{{ scope.row.title || '未命名' }}</span>
                  </template>
                </el-table-column>
                
                <el-table-column label="录音时间" prop="created_at" width="200">
                  <template #default="scope">
                    <span>{{ new Date(scope.row.created_at).toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai' }) }}</span>
                  </template>
                </el-table-column>

                <el-table-column label="状态" prop="status" width="120" />
                
                <el-table-column label="操作">
                  <template #default="scope">
                    <div v-if="scope.row.status === 'completed'" class="recording-actions">
                      <audio :src="scope.row.audio_url" controls></audio>
                      <div class="action-buttons">
                        
                        <el-tooltip 
                          v-if="canManageContent && tableViewMode === 'history'"
                          :content="scope.row.is_domain_featured ? '从精选移除' : '设为本篇精选'"
                          placement="top"
                        >
                          <el-button 
                            link 
                            @click="toggleDomainFeature(scope.row)"
                            :type="scope.row.is_domain_featured ? 'warning' : ''"
                          >
                            <el-icon><Trophy /></el-icon>
                          </el-button>
                        </el-tooltip>
                        
                        <template v-if="tableViewMode === 'featured'">
                          <el-button 
                            link 
                            @click="toggleLike(scope.row)" 
                            :type="scope.row.is_liked_by_me ? 'danger' : ''"
                            class="social-button"
                          >
                            <el-icon>
                              <transition name="el-fade-in" mode="out-in">
                                <StarFilled v-if="scope.row.is_liked_by_me" />
                                <Star v-else />
                              </transition>
                            </el-icon>
                            <span>{{ scope.row.likes_count > 0 ? scope.row.likes_count : '点赞' }}</span>
                          </el-button>
                          
                          <el-button 
                            link 
                            @click="openCommentDrawer(scope.row)"
                            class="social-button"
                          >
                            <el-icon><ChatDotRound /></el-icon>
                            <span>{{ scope.row.comments_count > 0 ? scope.row.comments_count : '评论' }}</span>
                          </el-button>
                        </template>

                        <template v-if="Number(scope.row.user_id) === Number(authStore.userId)">
                          <el-button link type="primary" @click="handleRenameRecording(scope.row)">重命名</el-button>
                          <el-button link type="danger" @click="handleDeleteRecording(scope.row)">删除</el-button>
                        </template>
                        
                      </div>
                    </div>
                    <span v-else>处理中...</span>
                  </template>
                </el-table-column>
                
                <el-table-column label="识别结果" width="150" align="center">
                  <template #default="scope">
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
                    <div v-else-if="scope.row.ai_status === 'processing'" class="status-indicator">
                      <el-icon class="is-loading"><Loading /></el-icon>
                      <span>正在识别...</span>
                    </div>
                    <div v-else-if="scope.row.ai_status === 'pending'" class="status-indicator">
                      <span>队列中...</span>
                    </div>
                    <div v-else-if="scope.row.ai_status === 'failed'">
                      <el-tag type="danger" size="small">识别失败</el-tag>
                      <el-button link type="primary" size="small" @click="retryTranscribe(scope.row)" style="margin-left: 5px;">重试</el-button>
                    </div>
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
        <el-empty v-else description="请从左侧选择一篇材料开始练习" />
      </div>

      <!-- 右侧 AI 助手 -->
      <el-button class="right-panel-toggle" circle @click="toggleRightPanel" :icon="MoreFilled" title="AI 助手" />
      <div class="right-panel" :class="{ collapsed: isRightPanelCollapsed }">
        <div class="panel-header">
          <h3>AI 助手</h3>
        </div>
        <div class="panel-content ai-assistant">
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
    
    <!-- 右键菜单 -->
    <div v-if="contextMenu.visible" :style="contextMenu.style" class="context-menu" @click="contextMenu.visible = false">
      <div class="menu-item" @click="handleRenameNode(contextMenu.node)">重命名</div>
      <div class="menu-item" @click="handleDeleteNode(contextMenu.node)">删除</div>
      <div v-if="contextMenu.node?.data.node_type === 'folder'" class="menu-separator"></div>
      <div v-if="contextMenu.node?.data.node_type === 'folder'" class="menu-item" @click="handleCreateNode('folder', contextMenu.node)">新建子文件夹</div>
      <div v-if="contextMenu.node?.data.node_type === 'folder'" class="menu-item" @click="handleCreateNode('text', contextMenu.node)">新建子文件</div>
    </div>

  </div>
  <el-drawer
    v-model="commentDrawer.visible"
    :title="`关于 “${commentDrawer.targetTitle}” 的评论`"
    direction="rtl"
    size="40%"
  >
    <div class="comment-panel" v-loading="commentDrawer.loading">
      <div class="comment-list">
        <el-empty v-if="!commentDrawer.comments.length" description="还没有评论，快来抢占沙发吧！"></el-empty>
        <!-- 【修正】:key="comment.ID" -> :key="comment.id" -->
        <div v-for="comment in commentDrawer.comments" :key="comment.id" class="comment-item">
          <div class="comment-header">
            <!-- 【修正】comment.User -> comment.user -->
            <router-link 
              :to="{ name: 'profile', params: { userId: comment.user.id } }" 
              class="username-link comment-author"
            >
              {{ comment.user.username }}
            </router-link>
            <!-- 【修正】comment.CreatedAt -> comment.created_at -->
            <span class="comment-time">{{ new Date(comment.CreatedAt).toLocaleString() }}</span>
          </div>
          <!-- 【修正】comment.Content -> comment.content -->
          <div class="comment-content">
            {{ comment.content }}
          </div>
        </div>
      </div>
      
      <div class="comment-input-area">
        <el-input 
          v-model="commentDrawer.newComment"
          type="textarea"
          placeholder="发表你的看法..."
          :rows="3"
        />
        <el-button type="primary" @click="submitComment" style="margin-top: 10px;">发表评论</el-button>
      </div>
    </div>
  </el-drawer>
</template>

<script setup>
// --- 导入部分 ---
import { ref, reactive, onMounted, onUnmounted, computed, nextTick, watch  } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import apiClient from '@/api';
import { ElMessageBox, ElMessage } from 'element-plus';
import RecordingControlBar from '@/components/RecordingControlBar.vue';
import {
  Folder, Document, FolderAdd, DocumentAdd, Search,
  ArrowLeft, ArrowRight, MoreFilled,  Loading, Trophy, Star,StarFilled, ChatDotRound
} from '@element-plus/icons-vue';
import { useAuthStore } from '@/stores/auth';

// =======================================================================
//                统一的评论抽屉 (Comment Drawer) 逻辑
// =======================================================================

// --- 1. 统一的状态管理对象 ---
const commentDrawer = reactive({
  visible: false,       // 控制抽屉的显示/隐藏
  loading: false,       // 控制评论列表的加载状态
  submitting: false,    // 控制提交按钮的加载状态
  
  targetType: null,     // 评论的目标类型: 'node' 或 'recording'
  targetId: null,       // 评论的目标ID
  targetTitle: '',      // 评论目标的标题，用于抽屉标题
  
  comments: [],         // 存放评论列表
  newComment: '',       // 新评论的输入内容
});


// --- 2. 统一的打开抽屉函数 ---
// 这个函数现在可以接收不同类型的对象 (node 或 recording)
const openCommentDrawer = async (target) => {
  // a. 判断评论的目标类型
  if (target.node_type) { // 假设 node 对象有 node_type 属性
    commentDrawer.targetType = 'node';
    commentDrawer.targetTitle = target.title;
  } else { // 否则认为是 recording 对象
    commentDrawer.targetType = 'recording';
    commentDrawer.targetTitle = target.title || '未命名录音';
  }
  commentDrawer.targetId = target.id;

  // b. 打开抽屉并加载数据
  commentDrawer.visible = true;
  commentDrawer.loading = true;
  
  try {
    // c. 根据类型构建不同的 API URL
    const apiUrl = commentDrawer.targetType === 'node'
      ? `/domain-nodes/${commentDrawer.targetId}/comments`
      : `/recordings/${commentDrawer.targetId}/comments`;
      
    const response = await apiClient.get(apiUrl);
    commentDrawer.comments = response.data;
  } catch (error) {
    ElMessage.error('加载评论失败');
  } finally {
    commentDrawer.loading = false;
  }
};


// --- 3. 统一的提交评论函数 ---
const submitComment = async () => {
  const content = commentDrawer.newComment.trim();
  if (!content || !commentDrawer.targetId) return;

  commentDrawer.submitting = true;
  try {
    // a. 根据类型构建不同的 API URL
    const apiUrl = commentDrawer.targetType === 'node'
      ? `/domain-nodes/${commentDrawer.targetId}/comments`
      : `/recordings/${commentDrawer.targetId}/comments`;

    const response = await apiClient.post(apiUrl, { content });
    
    // b. 乐观更新
    commentDrawer.comments.push(response.data);
    commentDrawer.newComment = '';

    // c. 【关键】更新主页面上显示的数量
    if (commentDrawer.targetType === 'node') {
      if (selectedNode.value && selectedNode.value.id === commentDrawer.targetId) {
          selectedNode.value.comments_count++;
      }
    } else { // 如果是 recording
      // 在精选列表里找到对应的录音并更新其评论数
      const recInList = featuredRecordingsForNode.value.find(r => r.id === commentDrawer.targetId);
      if (recInList) {
          recInList.comments_count++;
      }
    }
    
    ElMessage.success('评论成功！');
  } catch (error) {
    ElMessage.error('评论失败');
  } finally {
    commentDrawer.submitting = false;
  }
};

// --- 4. (可选) 抽屉关闭时清空状态 ---
const onCommentDrawerClose = () => {
  // 清空所有状态，为下次打开做准备
  commentDrawer.targetType = null;
  commentDrawer.targetId = null;
  commentDrawer.targetTitle = '';
  commentDrawer.comments = [];
  commentDrawer.newComment = '';
};

//历史录音
// 1. 引入 authStore 实例
// 【新增】设为/取消精选的函数

const toggleDomainFeature = async (recording) => {
  const targetState = !recording.is_domain_featured;
  const actionText = targetState ? '设为精选' : '取消精选';

  // 乐观更新 UI
  recording.is_domain_featured = targetState;

  try {
    await apiClient.post(`/recordings/${recording.id}/feature-in-domain`, {
      feature: targetState
    });
    ElMessage.success(`${actionText}成功！`);

    // 【重要】如果取消精选，需要将它从精选列表(如果已加载)中移除
    if (!targetState) {
        featuredRecordingsForNode.value = featuredRecordingsForNode.value.filter(r => r.id !== recording.id);
    }

  } catch (error) {
    // API 调用失败，回滚 UI
    recording.is_domain_featured = !targetState;
    ElMessage.error(`${actionText}失败`);
    console.error(`Failed to ${actionText}:`, error);
  }
};
const authStore = useAuthStore();

// 2. 添加核心状态
const tableViewMode = ref('history'); // 'history' 或 'featured'
const featuredRecordingsForNode = ref([]); // 存放当前节点的精选录音
const featuredLoading = ref(false);

// 3. 添加计算属性
const tableData = computed(() => {
  if (tableViewMode.value === 'history') {
    return nodeRecordings.value; // nodeRecordings 是你已有的历史录音数组
  } else {
    return featuredRecordingsForNode.value;
  }
});

// 4. 添加 API 调用函数
const fetchFeaturedForCurrentNode = async () => {
  if (!selectedNode.value) return;
  featuredLoading.value = true;
  try {
    const response = await apiClient.get(`/domains/${domainId}/nodes/${selectedNode.value.id}/featured-recordings`);
    featuredRecordingsForNode.value = response.data;
  } catch (error) {
    ElMessage.error('加载精选录音失败');
  } finally {
    featuredLoading.value = false;
  }
};

// 5. 添加事件处理和监听
const onViewModeChange = (newMode) => {
  if (newMode === 'featured') {
    fetchFeaturedForCurrentNode();
  }
};



// 7. 添加点赞函数
const toggleLike = async (recording) => {
    // 乐观更新 UI
    const originalState = recording.is_liked_by_me;
    recording.is_liked_by_me = !originalState;
    recording.likes_count += originalState ? -1 : 1;

    // 调用 API
    try {
        if (recording.is_liked_by_me) {
            await apiClient.post(`/recordings/${recording.id}/like`);
        } else {
            await apiClient.delete(`/recordings/${recording.id}/like`);
        }
    } catch (error) {
        // API 失败，回滚 UI
        recording.is_liked_by_me = originalState;
        recording.likes_count += originalState ? 1 : -1;
        ElMessage.error('操作失败');
    }
};


// --- 录音控制相关状态和逻辑 ---

// 1. 核心状态变量
const isUploading = ref(false);
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
  formData.append('domain_node_id', selectedNode.value.id);
  
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
// --- 路由与核心状态 ---
const route = useRoute();
const router = useRouter();
const domainId = route.params.id;
const domainInfo = ref(null);
const userRole = ref('member'); // 默认为普通成员
const loading = ref(true);

// --- 权限 ---
const canManageContent = computed(() => {
  return userRole.value === 'owner' || userRole.value === 'admin';
});

// --- 布局状态 ---
const isLeftPanelCollapsed = ref(false);
const isRightPanelCollapsed = ref(true);
const toggleLeftPanel = () => isLeftPanelCollapsed.value = !isLeftPanelCollapsed.value;
const toggleRightPanel = () => isRightPanelCollapsed.value = !isRightPanelCollapsed.value;

// --- 业务状态 ---
const treeRef = ref(null);
const treeLoading = ref(false);
const selectedNode = ref(null);
const nodeRecordings = ref([]);
const treeProps = { label: 'title', isLeaf: (data) => data.node_type === 'text' };
const contextMenu = reactive({ visible: false, style: { top: '0px', left: '0px' }, node: null });
const searchQuery = ref('');
const isSearching = ref(false);
const searchResults = ref([]);
const saveStatus = ref('');
const pollingTimers = new Map();
const MAX_POLLING_ATTEMPTS = 30;
const timedOutTaskIds = ref(new Set());
// --- AI 助手相关状态 ---
const chatMessages = ref([
    { role: 'assistant', content: '你好！有什么可以帮助你的吗？' }
]);
const userInput = ref('');
const isAiThinking = ref(false);
const chatScrollbarRef = ref(null);
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
// --- 帮助函数 ---
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

// --- API 调用与逻辑 ---

// 加载圈子节点
const loadNode = async (node, resolve) => {
  const parentId = node.level === 0 ? null : node.data.id;
  treeLoading.value = true;
  try {
    const response = await apiClient.get(`/domains/${domainId}/nodes`, { params: { parent_id: parentId } });
    resolve(response.data);
  } catch (error) {
    ElMessage.error('加载圈子内容失败');
    resolve([]);
  } finally {
    treeLoading.value = false;
  }
};

// 获取圈子节点的录音列表并管理轮询
const fetchRecordingsForCurrentNode = async () => {
    if (!selectedNode.value || selectedNode.value.node_type !== 'text') {
        clearAllPolls();
        return;
    }

    try {
        // 【关键】API 端点保持不变，仍然是 domain-nodes
        const response = await apiClient.get(`/domain-nodes/${selectedNode.value.id}/recordings`);
        const newRecordings = response.data;
        nodeRecordings.value = newRecordings; // 更新UI

        const activePollIds = new Set();

        newRecordings.forEach(rec => {
            let needsPolling = false;

            // --- 1. 监控主任务 (文件上传) ---
            // 注意：圈子录音上传后，我们假设其 status 很快会变为 'completed'
            // 如果后端对圈子录音也有复杂的 'processing' 状态，也需要像 MyContentPage 一样监控
            // 这里我们主要聚焦 AI 任务的监控
            if (rec.status === 'processing') {
                needsPolling = true;
                const timerInfo = pollingTimers.get(rec.id);
                if (timerInfo) {
                    timerInfo.uploadAttempts = (timerInfo.uploadAttempts || 0) + 1;
                    if (timerInfo.uploadAttempts > MAX_POLLING_ATTEMPTS) {
                        clearInterval(timerInfo.intervalId);
                        pollingTimers.delete(rec.id);
                        rec.status = 'failed';
                        ElMessage.error(`录音 ${rec.title || rec.id} 处理超时，请删除后重试。`);
                    }
                }
            }

            // --- 2. 监控 AI 任务 ---
            if (rec.status === 'completed' && (rec.ai_status === 'pending' || rec.ai_status === 'processing')) {
                needsPolling = true;
                const timerInfo = pollingTimers.get(rec.id);
                if (timerInfo) {
                    timerInfo.aiAttempts = (timerInfo.aiAttempts || 0) + 1;
                    if (timerInfo.aiAttempts > MAX_POLLING_ATTEMPTS * 2) { // AI 超时时间可以更长
                        clearInterval(timerInfo.intervalId);
                        pollingTimers.delete(rec.id);
                        rec.ai_status = 'failed';
                        ElMessage.warning(`录音 ${rec.title || rec.id} 的AI识别超时，但录音已保存，您可稍后重试识别。`);
                    }
                }
            }
            
            // --- 总控逻辑 ---
            if (needsPolling) {
                activePollIds.add(rec.id);
                if (!pollingTimers.has(rec.id)) {
                    startPolling(rec.id);
                }
            }
        });

        // 清理已完成任务的定时器
        for (const id of pollingTimers.keys()) {
            if (!activePollIds.has(id)) {
                clearInterval(pollingTimers.get(id).intervalId);
                pollingTimers.delete(id);
            }
        }

    } catch (error) {
        console.error("Failed to fetch domain node recordings", error);
    }
}

// 点击树节点
const handleNodeClick = async (data) => {
  clearAllPolls();
  tableViewMode.value = 'history'; // 在函数开头重置
  featuredRecordingsForNode.value = []; // 清空数据
  timedOutTaskIds.value.clear();
  
  selectedNode.value = data;
  if (data.node_type === 'text') {
    await fetchRecordingsForCurrentNode();
  }
};

// 右键菜单
const handleNodeContextMenu = (event, data, node) => {
  if (!canManageContent.value) return;
  event.preventDefault();
  contextMenu.style.left = event.clientX + 'px';
  contextMenu.style.top = event.clientY + 'px';
  contextMenu.node = node;
  contextMenu.visible = true;
};

// 拖拽规则
const allowDrop = (draggingNode, dropNode, type) => {
  if (!canManageContent.value) return false;
  if (dropNode.data.node_type !== 'folder') return false;
  if (type !== 'inner') return false;
  return true;
};

// 拖拽完成
const handleNodeDrop = async (draggingNode, dropNode) => {
  const draggingNodeId = draggingNode.data.id;
  const newParentId = dropNode.data.id;
  try {
    await apiClient.put(`/domains/${domainId}/nodes/${draggingNodeId}/move`, { new_parent_id: newParentId });
    ElMessage.success('移动成功！');
  } catch (error) {
    ElMessage.error('移动失败，将刷新页面以恢复。');
    setTimeout(() => window.location.reload(), 1500);
  }
};

// 内容管理 (仅管理员)
const handleCreateNode = (type, contextNode = null) => {
  ElMessageBox.prompt(`请输入新的${type === 'folder' ? '文件夹' : '文件'}名称`, '新建', { inputValidator: (val) => val && val.trim() !== '', inputErrorMessage: '名称不能为空' })
  .then(async ({ value }) => {
    const parentData = getNodeData(contextNode);
    let parentId = (parentData && parentData.node_type === 'folder') ? parentData.id : null;
    try {
      const response = await apiClient.post(`/domains/${domainId}/nodes`, { parent_id: parentId, node_type: type, title: value });
      const parentNodeInTree = parentId ? treeRef.value?.getNode(parentId) : null;
      treeRef.value?.append(response.data, parentNodeInTree);
      ElMessage.success('创建成功！');
    } catch (error) { ElMessage.error('创建失败'); }
  }).catch(() => {});
};

const handleRenameNode = (node) => {
  const nodeData = getNodeData(node);
  ElMessageBox.prompt('请输入新名称', '重命名', { inputValue: nodeData.title })
  .then(async ({ value }) => {
    if (!value || value.trim() === '' || value === nodeData.title) return;
    try {
      await apiClient.put(`/domains/${domainId}/nodes/${nodeData.id}`, { title: value });
      const nodeInTree = treeRef.value.getNode(nodeData.id);
      if (nodeInTree) nodeInTree.data.title = value;
      if (selectedNode.value?.id === nodeData.id) selectedNode.value.title = value;
      ElMessage.success('重命名成功');
    } catch (error) { ElMessage.error('重命名失败'); }
  }).catch(() => {});
};

const handleDeleteNode = (node) => {
  const nodeData = getNodeData(node);
  ElMessageBox.confirm(`确定要删除 "${nodeData.title}" 吗？`, '警告', { type: 'warning' })
  .then(async () => {
    try {
      await apiClient.delete(`/domains/${domainId}/nodes/${nodeData.id}`);
      treeRef.value.remove(nodeData.id);
      if (selectedNode.value?.id === nodeData.id) selectedNode.value = null;
      ElMessage.success('删除成功');
    } catch (error) { ElMessage.error('删除失败'); }
  }).catch(() => {});
};

// 内容保存
const saveContent = async () => {
  if (!selectedNode.value) return;
  saveStatus.value = '正在保存...';
  try {
    await apiClient.put(`/domains/${domainId}/nodes/${selectedNode.value.id}`, { content: selectedNode.value.content });
    saveStatus.value = '已保存';
  } catch (error) {
    saveStatus.value = '保存失败!'; ElMessage.error('保存失败');
  } finally { setTimeout(() => { saveStatus.value = ''; }, 2000); }
};
const debouncedSave = debounce(saveContent, 5000);
const onContentChange = () => { saveStatus.value = '内容已修改...'; debouncedSave(); };

// 搜索 (你需要一个后端的圈子搜索API)
const performSearch = async () => { ElMessage.info('圈子内搜索功能正在开发中...'); };
const cancelSearch = () => { isSearching.value = false; searchResults.value = []; };


// 录音管理
const handleRenameRecording = async (recording) => {
    ElMessageBox.prompt('请输入录音的新名称', '重命名录音', { inputValue: recording.title || '' })
    .then(async ({ value }) => {
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

// AI 识别与轮询
const retryTranscribe = async (recording) => {
    if (recording.ai_status === 'pending' || recording.ai_status === 'processing') return;

    try {
        // 更新UI状态为“队列中”
        recording.ai_status = 'pending';
        
        // 发送请求
        await apiClient.post(`/recordings/${recording.id}/transcribe`);
        ElMessage.success('已成功加入识别队列！');
        
        // 关键：立即启动一次轮询来更新状态和管理定时器
        fetchRecordingsForCurrentNode();

    } catch (error) {
        ElMessage.error('操作失败，请重试');
        // 如果请求失败，立即将状态回滚为 'failed'
        recording.ai_status = 'failed'; 
    }
};
const startPolling = (recordingId) => {
    if (pollingTimers.has(recordingId)) return;
    
    const intervalId = setInterval(fetchRecordingsForCurrentNode, 3000);
    // 初始化两个独立的计数器
    pollingTimers.set(recordingId, { 
        intervalId, 
        uploadAttempts: 0, 
        aiAttempts: 0 
    });
};
const clearAllPolls = () => {
    for (const timerInfo of pollingTimers.values()) {
        clearInterval(timerInfo.intervalId);
    }
    pollingTimers.clear();
};
const closeContextMenu = () => { contextMenu.visible = false; };

// --- 生命周期钩子 ---
onMounted(async () => {
  loading.value = true;
  try {
    const response = await apiClient.get(`/domains/${domainId}/details`);
    domainInfo.value = response.data.domain;
    userRole.value = response.data.role;
  } catch (error) {
    ElMessage.error(error.response?.data?.error || "无法加载圈子信息或您不是成员");
    router.push('/my-domains');
    return;
  } finally {
    loading.value = false;
  }
  document.addEventListener('click', closeContextMenu);
});

onUnmounted(() => {
  document.removeEventListener('click', closeContextMenu);
  clearAllPolls();
});

</script>

<style scoped>
/* 复制 MyContentPage.vue 的所有样式到这里 */
.readonly-content {
  white-space: pre-wrap;
  padding: 16px;
  font-size: 16px;
  line-height: 1.7;
  color: #333;
  background-color: #f9f9f9;
  border-radius: 5px;
  max-height: 40vh; /* 给一个最大高度，超出则滚动 */
  height: 100%;
  overflow-y: auto;
  min-height: 100px;
}
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
  flex-grow: 1; /* 占据所有可用空间 */
  min-height: 0; /* Flexbox 关键技巧，允许子元素内部滚动 */
  display: flex;
  flex-direction: column;
}

/* 3. 【解决问题一】让文本编辑区自动伸展 */
.editor-area {
  flex-grow: 1; /* 核心：占据所有剩余的垂直空间 */
  max-height: 1000px;
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

.history-list {
  flex-grow: 1; /* 占据录音区的剩余空间 */
  max-height: 280px;
  min-height: 0; /* 允许内部表格收缩 */
  /* overflow-y: auto;  <-- 这个可以移除了 */
  padding: 0 16px 16px 16px;
}
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
/* ==========================================================================
   保留您其他的样式 (头部、按钮组、右键菜单等)
   ========================================================================== */
.my-content-page, .content-layout-container, .left-panel, .right-panel,
.panel-header, .panel-content, .panel-search, .panel-actions, .panel-tree,
.tree-scrollbar, .custom-tree, :deep(.custom-tree .el-tree-node__content),
:deep(.custom-tree .el-tree-node.is-current > .el-tree-node__content),
.custom-tree-node .el-icon, .left-panel-toggle, .center-panel-header,
.filename, .actions-group, .save-status, .right-panel-toggle, .context-menu,
.menu-item, .menu-separator, .search-results, .search-result-item,
.search-empty, .ai-assistant, .chat-history, .chat-message, .message-bubble,
.loading-bubble, .chat-input, .recording-actions, .left-panel.collapsed, .right-panel.collapsed,
.left-panel-toggle.collapsed
 {
  /* 这里粘贴您提供的所有其他样式，确保它们保持不变 */
  /* 为节省篇幅，此处省略，请将您原有的、与核心布局无关的样式代码复制到这里 */
}

/* --- 您可以从这里开始，将您原有样式中除了上面已修正的部分，全部粘贴过来 --- */
/*
 *  【全新修正的样式】
 *  这部分代码实现了您描述的沉浸式布局
 */

.my-content-page {
  height: 100%;
  width: 100%;
  position: relative; /* 为绝对定位的折叠按钮提供上下文 */
}
.content-layout-container {
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
  margin-right: -16px; 
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


/* 左侧折叠按钮的定位 */
.left-panel-toggle {
  position: absolute;
  top: 50%;
  left: 284px; 
  transform: translateY(-50%);
  width: 24px;
  height: 48px;
  background-color: #f0f2f5;
  border: 1px solid #dcdfe6;
  border-left: none;
  border-top-right-radius: 6px;
  border-bottom-right-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  z-index: 10;
  transition: all 0.3s ease-in-out;
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
.right-panel-toggle { position: fixed; top: 76px; right: 24px; z-index: 100; }

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
.social-button {
  display: flex;
  align-items: center;
  gap: 4px; /* 图标和文字之间的间距 */
}

/* 评论抽屉样式 */
.comment-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
}
.comment-list {
  flex-grow: 1;
  overflow-y: auto;
  padding-bottom: 15px;
}
.comment-input-area {
  flex-shrink: 0;
  border-top: 1px solid #e4e7ed;
  padding-top: 15px;
}
.comment-item {
  margin-bottom: 20px;
  padding-bottom: 15px;
  border-bottom: 1px solid #f0f2f5;
}
.comment-header {
  display: flex;
  align-items: center;
  margin-bottom: 8px;
}
.comment-author {
  font-weight: 600;
  color: #333;
  margin-right: 10px;
}
.comment-time {
  font-size: 12px;
  color: #909399;
}
.comment-content {
  font-size: 14px;
  line-height: 1.6;
  color: #606266;
  white-space: pre-wrap; /* 保持换行 */
}
.username-link {
  color: #409eff; /* Element Plus 的主题蓝色 */
  text-decoration: none; /* 去掉下划线 */
  font-weight: 500;
  transition: color 0.2s;
}

.username-link:hover {
  color: #79bbff; /* 鼠标悬浮时变浅 */
  text-decoration: underline; /* 鼠标悬浮时显示下划线 */
}
.section-header {
  display: flex; /* 启用 Flexbox 布局 */
  justify-content: space-between; /* 核心：让左右两个 div 两端对齐 */
  align-items: center; /* 垂直居中 */
  padding: 0 16px; /* 左右留出一些边距，与表格对齐 */
  margin-bottom: 16px; /* 与表格之间的间距 */
  height: 40px; /* 给操作栏一个固定的高度 */
}
.left-controls {
  /* ... */
}
.right-controls {
  /* ... */
}
</style>
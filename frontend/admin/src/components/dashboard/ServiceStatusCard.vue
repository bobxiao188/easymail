<template>
  <div class="service-status-grid">
    <div
      v-for="service in services"
      :key="service.name"
      class="service-card"
      :class="`service-card--${service.status}`"
    >
      <div class="service-header">
        <div class="service-icon">
          <component :is="getStatusIcon(service.status)" />
        </div>
        <div class="service-info">
          <h3 class="service-name">{{ service.displayName }}</h3>
          <p class="service-description">{{ service.description }}</p>
        </div>
      </div>
      <div class="service-status-badge" :class="`status--${service.status}`">
        {{ getStatusText(service.status) }}
      </div>
      <div class="service-footer">
        <span class="heartbeat-time">{{ formatHeartbeat(service.lastHeartbeat) }}</span>
      </div>
    </div>

    <div v-if="services.length === 0" class="empty-state">
      <el-icon class="empty-icon"><Cpu /></el-icon>
      <p>{{ t('dashboard.noServices') }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Cpu, TrendCharts } from '@element-plus/icons-vue';
import { useI18n } from 'vue-i18n';
import type { ServiceStatus } from '../../api/dashboard';
const { t } = useI18n();
defineProps<{
 services: ServiceStatus[];
}>();
function getStatusIcon(status: string) {
 switch (status) {
 case 'running':
 return TrendCharts;
 case 'warning':
 return TrendCharts;
 case 'stopped':
 return TrendCharts;
 default:
 return TrendCharts;
 }
}
function getStatusText(status: string) {
 switch (status) {
 case 'running':
 return t('dashboard.statusRunning');
 case 'warning':
 return t('dashboard.statusWarning');
 case 'stopped':
 return t('dashboard.statusStopped');
 default:
 return t('dashboard.statusUnknown');
 }
}
function formatHeartbeat(timestamp: string) {
 if (!timestamp)
 return t('dashboard.noHeartbeat');
 const date = new Date(timestamp);
 const now = new Date();
 const diff = now.getTime() - date.getTime();
 const minutes = Math.floor(diff / 60000);
 if (minutes < 1)
 return t('dashboard.justNow');
 if (minutes < 60)
 return t('dashboard.minutesAgo', { count: minutes });
 const hours = Math.floor(minutes / 60);
 if (hours < 24)
 return t('dashboard.hoursAgo', { count: hours });
 return date.toLocaleDateString();
}
</script>

<style scoped>
.service-status-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 16px;
}

.service-card {
  background: var(--surface);
  border-radius: 12px;
  padding: 16px;
  border: 1px solid var(--border-default);
  transition: all 0.3s ease;
}

.service-card--running {
  border-color: var(--success);
}

.service-card--warning {
  border-color: var(--warning);
}

.service-card--stopped {
  border-color: var(--danger);
}

.service-card--unknown {
  border-color: var(--border-default);
}

.service-header {
  display: flex;
  gap: 12px;
  margin-bottom: 12px;
}

.service-icon {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.service-card--running .service-icon {
  background: rgba(34, 197, 94, 0.1);
  color: var(--success);
}

.service-card--warning .service-icon {
  background: rgba(251, 191, 36, 0.1);
  color: var(--warning);
}

.service-card--stopped .service-icon {
  background: rgba(239, 68, 68, 0.1);
  color: var(--danger);
}

.service-card--unknown .service-icon {
  background: rgba(156, 163, 175, 0.1);
  color: var(--foreground-muted);
}

.service-info {
  flex: 1;
  min-width: 0;
}

.service-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--foreground);
  margin: 0 0 4px 0;
}

.service-description {
  font-size: 12px;
  color: var(--foreground-muted);
  margin: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-status-badge {
  display: inline-block;
  padding: 4px 10px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 500;
  margin-bottom: 12px;
}

.status--running {
  background: rgba(34, 197, 94, 0.1);
  color: var(--success);
}

.status--warning {
  background: rgba(251, 191, 36, 0.1);
  color: var(--warning);
}

.status--stopped {
  background: rgba(239, 68, 68, 0.1);
  color: var(--danger);
}

.status--unknown {
  background: rgba(156, 163, 175, 0.1);
  color: var(--foreground-muted);
}

.service-footer {
  padding-top: 12px;
  border-top: 1px solid var(--border-default);
}

.heartbeat-time {
  font-size: 12px;
  color: var(--foreground-muted);
}

.empty-state {
  grid-column: 1 / -1;
  text-align: center;
  padding: 40px;
}

.empty-icon {
  font-size: 48px;
  color: var(--foreground-muted);
  margin-bottom: 16px;
}

.empty-state p {
  color: var(--foreground-muted);
  margin: 0;
}
</style>
<template>
  <div class="top-senders-table">
    <el-table
      :data="senders"
      :border="false"
      class="senders-table"
      :empty-text="t('dashboard.noTopSenders')"
    >
      <el-table-column
        :label="t('dashboard.rank')"
        width="60"
        align="center"
      >
        <template #default="{ row }">
          <span class="rank-badge" :class="getRankClass(row.rank)">
            {{ row.rank }}
          </span>
        </template>
      </el-table-column>
      <el-table-column
        :label="t('dashboard.sender')"
        prop="sender"
        min-width="200"
      >
        <template #default="{ row }">
          <div class="sender-info">
            <el-icon class="sender-icon">
              <component :is="row.type === 'domain' ? TrendCharts : TrendCharts" />
            </el-icon>
            <span class="sender-text">{{ row.sender }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column
        :label="t('dashboard.count')"
        prop="count"
        width="100"
        align="right"
      >
        <template #default="{ row }">
          <span class="count-value">{{ formatNumber(row.count) }}</span>
        </template>
      </el-table-column>
      <el-table-column
        :label="t('dashboard.percentage')"
        width="120"
        align="right"
      >
        <template #default="{ row }">
          <div class="percentage-cell">
            <div class="percentage-bar">
              <div
                class="percentage-fill"
                :style="{ width: getPercentage(row.count) + '%' }"
              ></div>
            </div>
            <span class="percentage-text">{{ getPercentage(row.count).toFixed(1) }}%</span>
          </div>
        </template>
      </el-table-column>
    </el-table>

    <div v-if="senders.length > 0" class="senders-summary">
      <div class="summary-item">
        <span class="summary-label">{{ t('dashboard.totalTopSenders') }}:</span>
        <span class="summary-value">{{ formatNumber(totalCount) }}</span>
      </div>
      <div class="summary-item">
        <span class="summary-label">{{ t('dashboard.average') }}:</span>
        <span class="summary-value">{{ formatNumber(Math.round(totalCount / senders.length)) }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">import { computed } from 'vue';
import { TrendCharts } from '@element-plus/icons-vue'; 
import { useI18n } from 'vue-i18n';
import type { TopSender } from '../../api/dashboard';
const { t } = useI18n();
const props = defineProps<{
 senders: TopSender[];
}>();
const totalCount = computed(() => {
 return props.senders.reduce((sum, sender) => sum + sender.count, 0);
});
function getRankClass(rank: number) {
 switch (rank) {
 case 1:
 return 'rank--gold';
 case 2:
 return 'rank--silver';
 case 3:
 return 'rank--bronze';
 default:
 return 'rank--default';
 }
}
function getPercentage(count: number) {
 if (totalCount.value === 0)
 return 0;
 return (count / totalCount.value) * 100;
}
function formatNumber(num: number) {
 if (num >= 10000) {
 return (num / 10000).toFixed(1) + 'w';
 }
 if (num >= 1000) {
 return (num / 1000).toFixed(1) + 'k';
 }
 return num.toString();
}
</script>

<style scoped>
.top-senders-table {
  width: 100%;
}

.senders-table {
  --el-table-bg-color: transparent;
  --el-table-tr-bg-color: transparent;
  --el-table-header-bg-color: transparent;
  --el-table-current-row-bg-color: transparent;
  --el-table-header-text-color: var(--foreground-muted);
  --el-table-row-hover-bg-color: var(--surface-hover);
}

.rank-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  font-size: 12px;
  font-weight: 600;
}

.rank--gold {
  background: linear-gradient(135deg, #fbbf24, #f59e0b);
  color: #fff;
}

.rank--silver {
  background: linear-gradient(135deg, #9ca3af, #6b7280);
  color: #fff;
}

.rank--bronze {
  background: linear-gradient(135deg, #d97706, #b45309);
  color: #fff;
}

.rank--default {
  background: var(--border-default);
  color: var(--foreground-muted);
}

.sender-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.sender-icon {
  font-size: 14px;
  color: var(--foreground-muted);
}

.sender-text {
  font-size: 13px;
  color: var(--foreground);
}

.count-value {
  font-size: 14px;
  font-weight: 600;
  color: var(--foreground);
}

.percentage-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.percentage-bar {
  flex: 1;
  max-width: 80px;
  height: 6px;
  background: var(--border-default);
  border-radius: 3px;
  overflow: hidden;
}

.percentage-fill {
  height: 100%;
  background: linear-gradient(90deg, var(--accent), var(--accent-light));
  border-radius: 3px;
  transition: width 0.3s ease;
}

.percentage-text {
  font-size: 12px;
  color: var(--foreground-muted);
  width: 40px;
  text-align: right;
}

.senders-summary {
  display: flex;
  justify-content: flex-end;
  gap: 24px;
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--border-default);
}

.summary-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.summary-label {
  font-size: 12px;
  color: var(--foreground-muted);
}

.summary-value {
  font-size: 14px;
  font-weight: 600;
  color: var(--foreground);
}
</style>
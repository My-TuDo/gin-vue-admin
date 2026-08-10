<script setup>
import { onLaunch } from '@dcloudio/uni-app'

onLaunch(() => {
  console.log('[mini] App Launch')
})
</script>

<style lang="scss">
/* =========================================================
 * 全局公共样式（设计令牌来自 uni.scss，编译自动注入）
 * 语义化 class 供所有页面复用；页面私有类只保留布局差异
 * ========================================================= */

/* 页面基础 */
page {
  background-color: $ui-bg-page;
  font-size: $ui-font-base;
  color: $ui-text-1;
  -webkit-font-smoothing: antialiased;
}

/* ---------- 按压反馈（配合 view hover-class） ---------- */
.ui-hover {
  opacity: 0.85;
  transition: opacity 0.15s;
}

/* ---------- 页面容器 ---------- */
.ui-page {
  min-height: 100vh;
  padding: $ui-space-md;
  box-sizing: border-box;
}

/* ---------- 白卡片 ---------- */
.ui-card {
  background: $ui-bg-card;
  border-radius: $ui-radius-md;
  padding: $ui-space-md;
  box-shadow: $ui-shadow-card;
  box-sizing: border-box;
}

/* ---------- 列表行（flex 两端对齐 + 底部分割线） ---------- */
.ui-cell {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: $ui-space-sm 0;
  border-bottom: 1rpx solid $ui-border;

  &:last-child {
    border-bottom: none;
  }
}

/* ---------- 区块标题（左侧主色竖条） ---------- */
.ui-section-title {
  display: flex;
  align-items: center;
  font-size: $ui-font-md;
  font-weight: 600;
  color: $ui-text-1;
  margin: $ui-space-xs 0 $ui-space-sm;

  &::before {
    content: '';
    width: 6rpx;
    height: 28rpx;
    border-radius: 3rpx;
    background: $ui-primary;
    margin-right: 12rpx;
    flex-shrink: 0;
  }
}

/* ---------- 徽标（浅底深字胶囊） ---------- */
.ui-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: $ui-font-xs;
  font-weight: 600;
  padding: 4rpx 14rpx;
  border-radius: $ui-radius-full;
  flex-shrink: 0;
}
.ui-badge--success { background: $ui-primary-bg; color: $ui-primary; }
.ui-badge--warning { background: $ui-warning-bg; color: $ui-warning-text; }
.ui-badge--danger { background: $ui-danger-bg; color: $ui-danger; }
.ui-badge--info { background: $ui-bg-disabled; color: $ui-text-2; }
.ui-badge--blue { background: $ui-blue-bg; color: $ui-blue; }
.ui-badge--primary { background: $ui-primary-bg; color: $ui-primary; }

/* ---------- 按钮（主/描边/禁用 + 按压态） ---------- */
.ui-btn-primary,
.ui-btn-ghost,
.ui-btn-disabled {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  box-sizing: border-box;
  font-weight: 600;
  transition: opacity 0.15s;

  &::after {
    border: none;
  }

  &:active {
    opacity: 0.8;
  }
}
.ui-btn-primary {
  background: $ui-primary;
  color: $ui-bg-card;
  border-radius: $ui-radius-full;
  font-size: $ui-font-base;
  min-height: $ui-touch;
  padding: 0 40rpx;
}
.ui-btn-ghost {
  background: $ui-bg-card;
  color: $ui-primary;
  border-radius: $ui-radius-full;
  border: 1rpx solid $ui-primary;
  font-size: $ui-font-base;
  min-height: $ui-touch;
  padding: 0 40rpx;

  &--danger {
    color: $ui-danger;
    border-color: $ui-danger;
    background: $ui-bg-card;
  }
}
.ui-btn-disabled {
  background: $ui-bg-disabled;
  color: $ui-text-3;
  border-radius: $ui-radius-full;
  font-size: $ui-font-base;
  min-height: $ui-touch;
  padding: 0 40rpx;
}

/* ---------- 空态（CSS 简笔图标 + 主/副文案） ---------- */
.ui-empty {
  position: relative;
  padding: 60rpx 0 80rpx;
  text-align: center;
  color: $ui-text-3;
  font-size: $ui-font-sm;

  &::before {
    content: '';
    display: block;
    width: 64rpx;
    height: 64rpx;
    margin: 0 auto 20rpx;
    border: 4rpx solid $ui-text-3;
    border-radius: $ui-radius-md;
    opacity: 0.45;
    box-sizing: border-box;
  }

  &::after {
    content: '';
    position: absolute;
    top: 92rpx;
    left: 50%;
    width: 32rpx;
    height: 4rpx;
    margin-left: -16rpx;
    background: $ui-text-3;
    border-radius: 2rpx;
    opacity: 0.45;
  }
}
.ui-empty-main {
  display: block;
  font-size: $ui-font-sm;
  color: $ui-text-3;
}
.ui-empty-sub {
  display: block;
  margin-top: 8rpx;
  font-size: $ui-font-xs;
  color: $ui-text-3;
  opacity: 0.85;
}

/* ---------- 错误占位 ---------- */
.ui-err {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 80rpx 0;
}
.ui-err-text {
  font-size: $ui-font-sm;
  color: $ui-text-3;
  margin-bottom: $ui-space-sm;
}
.ui-retry {
  display: inline-block;
  padding: 12rpx 40rpx;
  background: $ui-primary;
  color: $ui-bg-card;
  font-size: $ui-font-sm;
  border-radius: $ui-radius-full;
  font-weight: 600;

  &::after {
    border: none;
  }

  &:active {
    opacity: 0.8;
  }
}

/* ---------- 数量步进器（按钮 72rpx 圆角方、中缝数字） ---------- */
.ui-stepper {
  display: flex;
  align-items: center;
  gap: 8rpx;
  background: $ui-bg-page;
  border-radius: $ui-radius-sm;
  padding: 4rpx 6rpx;
  flex-shrink: 0;

  .step-btn {
    width: 72rpx;
    height: 72rpx;
    line-height: 72rpx;
    text-align: center;
    background: $ui-bg-card;
    border-radius: 12rpx;
    font-size: 30rpx;
    font-weight: 700;
    color: $ui-text-1;
    box-shadow: 0 2rpx 6rpx rgba(0, 0, 0, 0.06);

    &.plus {
      color: $ui-primary;
    }

    &.disabled {
      color: $ui-text-3;
      background: $ui-bg-hover;
      opacity: 0.6;
    }
  }

  .step-num {
    min-width: 64rpx;
    text-align: center;
    font-size: $ui-font-base;
    font-weight: 700;
    color: $ui-text-1;
  }
}

/* ---------- 底部弹层（遮罩淡入 + 白底上圆角弹体） ---------- */
.ui-pop-mask {
  position: fixed;
  inset: 0;
  z-index: $ui-z-mask;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  flex-direction: column;
  justify-content: flex-end;
  animation: uiFadeIn 0.2s ease;
}
.ui-pop-body {
  background: $ui-bg-card;
  border-radius: $ui-radius-lg $ui-radius-lg 0 0;
  max-height: 70vh;
  display: flex;
  flex-direction: column;
  padding-bottom: env(safe-area-inset-bottom);
  animation: uiPopUp 0.25s $ui-ease-out;
}
@keyframes uiFadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}
@keyframes uiPopUp {
  from { transform: translateY(100%); }
  to { transform: translateY(0); }
}

/* ---------- 居中弹窗变体（dialog，遮罩淡入 + 内容缩放淡入） ---------- */
.ui-pop-mask--center {
  justify-content: center;

  .ui-pop-body {
    width: 600rpx;
    max-height: 80vh;
    border-radius: $ui-radius-lg;
    padding: 40rpx 36rpx 32rpx;
    box-sizing: border-box;
    animation: uiPopIn 0.2s $ui-ease-out;
  }
}
@keyframes uiPopIn {
  from { opacity: 0; transform: scale(0.96); }
  to { opacity: 1; transform: scale(1); }
}

/* ---------- 搜索框（灰底胶囊 + 左侧放大镜 + 右侧动作） ---------- */
.ui-search {
  position: relative;
  display: flex;
  align-items: center;
  background: $ui-bg-page;
  border-radius: $ui-radius-full;
  padding: 8rpx 8rpx 8rpx 44rpx;
  box-sizing: border-box;

  &::before {
    content: '';
    position: absolute;
    left: 20rpx;
    top: 50%;
    transform: translateY(-60%);
    width: 20rpx;
    height: 20rpx;
    border: 3rpx solid $ui-text-3;
    border-radius: 50%;
    box-sizing: border-box;
  }

  &::after {
    content: '';
    position: absolute;
    left: 39rpx;
    top: calc(50% + 9rpx);
    width: 3rpx;
    height: 12rpx;
    background: $ui-text-3;
    transform: rotate(-45deg);
  }

  .ui-search-input {
    flex: 1;
    height: 88rpx;
    line-height: 88rpx;
    min-width: 0;
    font-size: $ui-font-sm;
    color: $ui-text-1;
  }

  .ui-search-action {
    margin-left: $ui-space-sm;
    height: 88rpx;
    line-height: 88rpx;
    padding: 0 28rpx;
    background: $ui-primary;
    color: $ui-bg-card;
    font-size: $ui-font-sm;
    font-weight: 600;
    border-radius: $ui-radius-full;
    flex-shrink: 0;
  }
}
</style>

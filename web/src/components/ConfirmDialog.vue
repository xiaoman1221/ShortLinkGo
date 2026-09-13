<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="modelValue" class="overlay" @click.self="close">
        <div class="dialog" role="alertdialog" aria-modal="true">
          <h3>{{ title }}</h3>
          <p>{{ message }}</p>
          <div class="actions">
            <button class="btn btn-ghost btn-sm" type="button" @click="close">取消</button>
            <button class="btn btn-solid-danger btn-sm" type="button" @click="confirm">删除</button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
const props = defineProps({
  modelValue: { type: Boolean, default: false },
  title: { type: String, default: '确认操作' },
  message: { type: String, default: '' }
})
const emit = defineEmits(['update:modelValue', 'confirm'])

function close() {
  emit('update:modelValue', false)
}
function confirm() {
  emit('confirm')
  close()
}
</script>

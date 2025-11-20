package components

import (
	"testing"
)

func TestSquashAnimationComponent_GetProgress(t *testing.T) {
	tests := []struct {
		name        string
		elapsedTime float64
		duration    float64
		want        float64
	}{
		{
			name:        "开始时进度为0",
			elapsedTime: 0.0,
			duration:    0.7,
			want:        0.0,
		},
		{
			name:        "中间进度",
			elapsedTime: 0.35,
			duration:    0.7,
			want:        0.5,
		},
		{
			name:        "结束时进度为1",
			elapsedTime: 0.7,
			duration:    0.7,
			want:        1.0,
		},
		{
			name:        "超过时长进度仍为1",
			elapsedTime: 1.0,
			duration:    0.7,
			want:        1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SquashAnimationComponent{
				ElapsedTime: tt.elapsedTime,
				Duration:    tt.duration,
			}
			got := s.GetProgress()
			if got != tt.want {
				t.Errorf("GetProgress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSquashAnimationComponent_IsComplete(t *testing.T) {
	tests := []struct {
		name        string
		elapsedTime float64
		duration    float64
		isCompleted bool
		want        bool
	}{
		{
			name:        "未完成",
			elapsedTime: 0.5,
			duration:    0.7,
			isCompleted: false,
			want:        false,
		},
		{
			name:        "刚好完成",
			elapsedTime: 0.7,
			duration:    0.7,
			isCompleted: false,
			want:        true,
		},
		{
			name:        "已标记完成",
			elapsedTime: 0.0,
			duration:    0.7,
			isCompleted: true,
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SquashAnimationComponent{
				ElapsedTime: tt.elapsedTime,
				Duration:    tt.duration,
				IsCompleted: tt.isCompleted,
			}
			if got := s.IsComplete(); got != tt.want {
				t.Errorf("IsComplete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSquashAnimationComponent_GetCurrentFrameIndex(t *testing.T) {
	// 创建测试用的帧数据（8帧）
	frames := make([]LocatorFrame, 8)
	for i := range frames {
		frames[i] = LocatorFrame{X: float64(i * 10)}
	}

	tests := []struct {
		name        string
		elapsedTime float64
		duration    float64
		want        int
	}{
		{
			name:        "第一帧",
			elapsedTime: 0.0,
			duration:    0.8,
			want:        0,
		},
		{
			name:        "中间帧",
			elapsedTime: 0.4,
			duration:    0.8,
			want:        4,
		},
		{
			name:        "最后一帧",
			elapsedTime: 0.8,
			duration:    0.8,
			want:        7,
		},
		{
			name:        "超过时长返回最后一帧",
			elapsedTime: 1.0,
			duration:    0.8,
			want:        7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SquashAnimationComponent{
				ElapsedTime:   tt.elapsedTime,
				Duration:      tt.duration,
				LocatorFrames: frames,
			}
			if got := s.GetCurrentFrameIndex(); got != tt.want {
				t.Errorf("GetCurrentFrameIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

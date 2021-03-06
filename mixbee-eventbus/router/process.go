

/***************************************************
Copyright 2016 https://github.com/AsynkronIT/protoactor-go

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*****************************************************/
package router

import (
	"sync"
	"sync/atomic"

	"github.com/mixbee/mixbee/mixbee-eventbus/actor"
)

// process serves as a proxy to the router implementation and forwards messages directly to the routee. This
// optimization avoids serializing router messages through an actor
type process struct {
	router   *actor.PID
	state    Interface
	mu       sync.Mutex
	watchers actor.PIDSet
	stopping int32
}

func (ref *process) SendUserMessage(pid *actor.PID, message interface{}) {
	_, msg, _ := actor.UnwrapEnvelope(message)
	if _, ok := msg.(ManagementMessage); !ok {
		ref.state.RouteMessage(message)
	} else {
		r, _ := actor.ProcessRegistry.Get(ref.router)
		// Always send the original message to the router actor,
		// since if the message is enveloped, the sender need to get a response.
		r.SendUserMessage(pid, message)
	}
}

func (ref *process) SendSystemMessage(pid *actor.PID, message interface{}) {
	switch msg := message.(type) {
	case *actor.Watch:
		if atomic.LoadInt32(&ref.stopping) == 1 {
			if r, ok := actor.ProcessRegistry.Get(msg.Watcher); ok {
				r.SendSystemMessage(msg.Watcher, &actor.Terminated{Who: pid})
			}
			return
		}
		ref.mu.Lock()
		ref.watchers.Add(msg.Watcher)
		ref.mu.Unlock()

	case *actor.Unwatch:
		ref.mu.Lock()
		ref.watchers.Remove(msg.Watcher)
		ref.mu.Unlock()

	case *actor.Stop:
		term := &actor.Terminated{Who: pid}
		ref.mu.Lock()
		ref.watchers.ForEach(func(_ int, other actor.PID) {
			if r, ok := actor.ProcessRegistry.Get(&other); ok {
				r.SendSystemMessage(&other, term)
			}
		})
		ref.mu.Unlock()

	default:
		r, _ := actor.ProcessRegistry.Get(ref.router)
		r.SendSystemMessage(pid, message)

	}
}

func (ref *process) Stop(pid *actor.PID) {
	if atomic.SwapInt32(&ref.stopping, 1) == 1 {
		return
	}

	ref.router.StopFuture().Wait()
	actor.ProcessRegistry.Remove(pid)
	ref.SendSystemMessage(pid, &actor.Stop{})
}

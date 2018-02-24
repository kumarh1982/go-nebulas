// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// the go-nebulas library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with the go-nebulas library.  If not, see <http://www.gnu.org/licenses/>.
//

package dag

import (
	"sync"

	"github.com/nebulasio/go-nebulas/util/logging"
)

type Callback func(*Node) error

// Task struct
type Task struct {
	dependence int
	node       *Node
}

// Dispatcher struct a message dispatcher dag.
type Dispatcher struct {
	concurrency int
	cb          Callback
	muTask      sync.Mutex
	dag         *Dag
	quitCh      chan bool
	queueCh     chan *Node
	tasks       map[string]*Task
	cursor      int
	err         error
}

// NewDispatcher create Dag Dispatcher instance.
func NewDispatcher(dag *Dag, concurrency int, cb Callback) *Dispatcher {
	dp := &Dispatcher{
		concurrency: concurrency,
		dag:         dag,
		cb:          cb,
		tasks:       make(map[string]*Task, 0),
		quitCh:      make(chan bool, 10),
		queueCh:     make(chan *Node, 100),
		cursor:      0,
	}
	return dp
}

// Run dag dispatch goroutine.
func (dp *Dispatcher) Run() error {
	logging.CLog().Info("Starting Dag Dispatcher...")

	vertices := dp.dag.GetNodes()

	for _, node := range vertices {
		task := &Task{
			dependence: node.ParentCounter,
			node:       node,
		}
		task.dependence = node.ParentCounter
		dp.tasks[node.Key] = task

		if task.dependence == 0 {
			dp.push(node)
		}
	}

	dp.loop()

	return dp.err
}

// loop
func (dp *Dispatcher) loop() {
	logging.CLog().Info("loop Dag Dispatcher.")

	//timerChan := time.NewTicker(time.Second).C
	wg := new(sync.WaitGroup)
	wg.Add(dp.concurrency)

	for i := 0; i < dp.concurrency; i++ {
		//logging.CLog().Info("loop Dag Dispatcher i:", i)
		go func(i int) {
			defer wg.Done()
			for {
				select {
				//case <-timerChan:
				//fmt.Printf("====numGo:==%d i=%d\n", runtime.NumGoroutine(), i)
				//metricsDispatcherCached.Update(int64(len(dp.receivedMessageCh)))
				case <-dp.quitCh:
					logging.CLog().Info("Stoped Dag Dispatcher.")
					return
				case msg := <-dp.queueCh:
					// callback todo
					node := msg
					err := dp.cb(node)

					if err != nil {
						dp.err = err
						dp.Stop()
						return
					}
					dp.CompleteParentTask(msg)
				}
			}
		}(i)
	}

	wg.Wait()
}

// Stop stop goroutine.
func (dp *Dispatcher) Stop() {
	logging.CLog().Info("Stopping dag Dispatcher...")

	for i := 0; i < dp.concurrency; i++ {
		dp.quitCh <- true
	}
}

// push queue channel
func (dp *Dispatcher) push(vertx *Node) {
	dp.queueCh <- vertx
}

// CompleteParentTask completed parent tasks
func (dp *Dispatcher) CompleteParentTask(node *Node) {
	dp.muTask.Lock()
	defer dp.muTask.Unlock()

	key := node.Key

	vertices := dp.dag.GetChildrenNodes(key)
	for _, node := range vertices {
		dp.updateDependenceTask(node.Key)
	}

	dp.cursor++

	if dp.cursor == dp.dag.Len() {
		//fmt.Println("cursor:", dp.cursor, " key:", key)
		dp.Stop()
	}
}

// updateDependenceTask task counter
func (dp *Dispatcher) updateDependenceTask(key string) {
	if _, ok := dp.tasks[key]; ok {
		dp.tasks[key].dependence--
		//fmt.Println("Key:", key, " dependence:", dp.tasks[key].dependence)
		if dp.tasks[key].dependence == 0 {
			dp.push(dp.tasks[key].node)
		}
	}
}

package curvecp

import (
	"math/rand"
	"time"
)

// Scheduler keeps track of the congestion of the link and adjusts
// various parameters, the most important two being the
// inter-transmission delay and the retransmit timer.
//
// This is an implementation of CurveCP's Chicago congestion control
// algorithm... At least I hope it's a faithful implementation, I'm
// trying to untangle the reference implementation.
type scheduler struct {
	// Interval between successive transmissions. This increases and
	// drops off in a cycle dictated by network congestion.
	txThrottle time.Duration
	// Retransmit timeout. Packets not acknowledged for this amount of
	// time will be retransmitted, potentially triggering other
	// behaviors like panicking and drastically reducing txThrottle.
	txTimeout time.Duration

	// Random source for jittering. NOT a crypto-safe source.
	rand rand.Rand

	// rttAverage and rttMeanDev form the Jacobson/Karels RTT
	// estimator, described in appendix A of
	// http://ee.lbl.gov/papers/congavoid.pdf . This estimator drives
	// the value of txTimeout, taking into account both the average
	// observed RTT and RTT variance.
	rttAverage time.Duration
	rttMeanDev time.Duration

	// rttHigh and rttLow are simpler estimators of the highest and
	// lowest RTTs seen in the lifetime of the connection. They
	// deliberately converge slower than the Jacobson/Karels estimator
	// above so as to more accurately track the highs and lows.
	rttHigh time.Duration
	rttLow  time.Duration

	// The last time we adjusted txThrottle
	lastThrottleAdjustment time.Time

	// Whether we saw a high or low in the previous adjustment
	// cycle. This keeps track of whether we just reached the
	// top/bottom of the congestion cycle.
	wasHigh, wasLow bool
	// True if we're in the falling part of the congestion cycle, true
	// if we're in the rising part.
	falling bool
	// Last time we reversed direction in the congestion cycle.
	lastEdge time.Time
	// Last time we doubled the transmission rate.
	lastDoubling time.Time
}

func newScheduler() *scheduler {
	return &scheduler{
		txThrottle: time.Second,
		txTimeout:  time.Second,
		rand:       rand.New(rand.NewSource(time.Now.UnixNano())),
	}
}

func (s *scheduler) init(initRtt time.Duration) {
	s.txThrottle = rtt
	s.rttAverage = rtt
	s.rttDeviation = rtt / 2
	s.rttHigh = rtt
	s.rttLow = rtt
	s.lastThrottleAdjustment = time.Now()
}

// Adjust adjusts the scheduler variables based on a new observation
// of RTT.
func (s *scheduler) Adjust(rtt time.Duration) {
	// If this is the first RTT observation, initialize the
	// scheduler.
	if s.rttAverage == 0 {
		s.init(rtt)
	}

	// This is Jacobson/Karels's txTimeout calculation, straight from
	// the paper, with a gain of .125 for the average and .25 for
	// deviation. The paper goes into great detail about this
	// algorithm, refer to it for more.
	averageDelta := rtt - s.rttAverage
	s.rttAverage += averageDelta / 8
	meanDevDelta := abs(averageDelta) - s.rttMeanDev
	s.rttMeanDev += meanDevDelta / 4
	s.txTimeout = s.rttAverage + 4*s.rttDeviation
	// The reference implementation throws in more delay here to
	// account for delayed acks. Not sure why.
	s.txTimeout += 8 * s.txThrottle

	// Adjust the top and bottom of the congestion cycle.
	s.rttHigh += (rtt - s.rttHigh) / 1024
	lowDelta := rtt - s.rttLow
	// Adjust the low mark upwards slower than we adjust it downwards.
	if lowDelta > 0 {
		s.rttLow += lowDelta / 8192
	} else {
		s.rttLow += lowDelta / 256
	}

	sinceAdjust := time.Since(s.lastThrottleAdjustment)
	// Reconsider txThrottle every 16 packet intervals.
	if sinceAdjust >= 16*s.txThrottle {
		if sinceAdjust > 10*time.Second {
			// No activity for >10s, do a slow restart.
			s.txThrottle = time.Second + s.rand.int63n(int64(time.Second/8))
		}

		s.lastThrottleAdjustment = time.Now()

		// Additive increase to the transmission rate, if we're not
		// already at ludicrous speed.
		if s.txThrottle > 100*time.Microsecond {
			// These maths taken straight from the implementation. I'm
			// not sure I get why it adjusts the way it does, but I
			// want to make progress.
			const timeConstant = 2251799813685248 // 2^51
			if s.txThrottle < 16*time.Millisecond {
				// N = N - cN^3, avoids 1 division and is close to the next curve.
				s.txThrottle -= s.txThrottle * s.txThrottle * s.txThrottle / timeConstant
			} else {
				// N = N/(1 + cN^2)
				s.txThrottle = s.txThrottle / (1 + s.txThrottle*s.txThrottle/timeConstant)
			}
		}

		if s.falling {
			// If we're at the low point of the cycle again, start
			// watching for the high point once more.
			if s.wasLow {
				s.falling = false
			}
		} else {
			// We're past the high point of congestion, back off.
			if s.wasHigh {
				s.txThrottle += s.rand.Int63n(int64(s.txThrottle/4))
				s.lastEdge = time.Now()
				s.falling = true
			}
		}

		s.wasLow = s.rttAverage < s.rttLow
		s.wasHigh = s.rttAverage > (rttHigh + 5*time.Millisecond)

		// Occasionally double our send rate, if not already at
		// ludicrous speed.
	double:
		if s.txThrottle > 100*time.Microsecond {
			if time.Since(s.lastEdge) < 60*time.Second {
				if time.Now.Before(s.lastDoubling + (4 * s.txThrottle) + (64 * s.txTimeout) + (5*time.Second)) {
					break double
				}
			} else {
				if time.Now.Before(s.lastDoubling + (4 * s.txThrottle) + (2*s.txTimeout)) {
					break double
				}
			}

			s.txThrottle /= 2
			lastDoubling = time.Now()
			lastEdge = time.Now()
		}
	}
}

func abs(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

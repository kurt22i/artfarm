package lib

import (
	"errors"
	"fmt"
)

func (g *Generator) RandMain(s SlotType) (StatType, error) {

	if s == Flower {
		return HP, nil
	}
	if s == Feather {
		return ATK, nil
	}

	var total, sum float64
	for _, v := range MainStatWeight[s] {
		total += v
	}

	p := g.r.Float64()

	for i, v := range MainStatWeight[s] {
		sum += v / total
		if p <= sum {
			return StatType(i), nil
		}
	}

	return 0, errors.New("error generating new stat - none found")
}

func (g *Generator) RandSubs(m StatType, lvl int) [][]float64 {
	//first element is total, rest is each roll

	var sum, total, p float64
	next := make([]float64, EndStatType)
	subIndex := make([]int, 4)
	result := make([][]float64, EndStatType)
	for i := 0; i < int(EndStatType); i++ {
		result[i] = make([]float64, 1, 6)
	}

	lines := 3
	if g.r.Float64() > .8 {
		lines = 4
		//log.Println("4 lines")
	}

	//total for initial sub
	for i, v := range SubWeights {
		if i == int(m) {
			continue
		}
		total += v
		next[i] = v
	}

	//if artifact lvl is less than 4 AND lines =3, then we only want to roll 3 substats
	n := 4
	if lvl < 4 && lines < 4 {
		n = 3
	}

	var found bool
	for i := 0; i < n; i++ {
		//log.Println(next, sum, total)
		p = g.r.Float64()
		for j, v := range next {
			sum += v / total
			if !found && p <= sum {
				result[j][0] = SubTier[j][g.r.Intn(4)]
				result[j] = append(result[j], result[j][0])
				subIndex[i] = j
				found = true
				//zero out weight for this sub
				next[j] = 0
			}
		}

		//add up total for next run
		sum = 0
		total = 0
		found = false
		for _, v := range next {
			total += v
		}
	}

	up := lvl / 4

	if lines == 3 {
		up--
	}

	for i := 0; i < up; i++ {
		pick := g.r.Intn(4)
		tier := g.r.Intn(4)
		result[subIndex[pick]][0] += SubTier[subIndex[pick]][tier]
		result[subIndex[pick]] = append(result[subIndex[pick]], SubTier[subIndex[pick]][tier])
	}

	return result
}

func (g *Generator) RandSubsNoHist(m StatType, lvl int) []float64 {
	//first element is total, rest is each roll

	var sum, total, p float64
	next := make([]float64, EndStatType)
	subIndex := make([]int, 4)
	result := make([]float64, EndStatType)

	lines := 3
	if g.r.Float64() > .8 {
		lines = 4
		//log.Println("4 lines")
	}

	//total for initial sub
	for i, v := range SubWeights {
		if i == int(m) {
			continue
		}
		total += v
		next[i] = v
	}

	//if artifact lvl is less than 4 AND lines =3, then we only want to roll 3 substats
	n := 4
	if lvl < 4 && lines < 4 {
		n = 3
	}

	var found bool
	for i := 0; i < n; i++ {
		//log.Println(next, sum, total)
		p = g.r.Float64()
		for j, v := range next {
			sum += v / total
			if !found && p <= sum {
				result[j] = SubTier[j][g.r.Intn(4)]
				subIndex[i] = j
				found = true
				//zero out weight for this sub
				next[j] = 0
			}
		}

		//add up total for next run
		sum = 0
		total = 0
		found = false
		for _, v := range next {
			total += v
		}
	}

	up := lvl / 4

	if lines == 3 {
		up--
	}

	for i := 0; i < up; i++ {
		pick := g.r.Intn(4)
		tier := g.r.Intn(4)
		result[subIndex[pick]] += SubTier[subIndex[pick]][tier]
	}

	return result
}

func PrintSubs(in [][]float64) {
	for i, v := range in {
		if v[0] == 0 {
			continue
		}
		fmt.Printf("%v: %.4f, rolls: %v\n", StatTypeString[i], v[0], v[1:])
	}
}

/*type Set struct {
	Flower Artifact
	Feather Artifact
	Sands Artifact
	Goblet Artifact
	Circlet Artifact
}*/

type Artifact struct {
	Slot SlotType
	Main StatType
	Subs []float64
	Ok   bool
	Set  int
}

const maxTries = 1000000000 //1 bil

//FarmArtifact return number of tries it tooks to reach the desired subs
//main is the desired main stat; if main == EndStatType then there's no requirement
func (g *Generator) FarmArtifact(main [][lib.EndSlotType]lib.StatType, desired [][lib.EndStatType]float64, set [][]int, maxdomain int ) (int, error) {
	var err error
	var req, score float64
	count := 0
	//bag := make([]Artifact, EndSlotType)
	keep := 5; the number of on and off pieces to store for optimizing per slot per char
	onpieces := [4][5][keep]Artifact; //first [] is which char, second is what type it is, third is which of the 5 stored artis it is
	offpieces := [4][5][keep]Artifact;
	onmin := [4][5]float64; //first [] is which char, second is is what type. stores the score the arti that is most replacable.
	onloc := [4][5]int; //first [] is which char, second is is what type. stores which of the 5 artis in this slot are least good/most replacable
	onmap := [4][5][keep]float64; //stores the score of all the pieces
	offmin := [4][5]float64; //first [] is which char, second is is what type. stores the score the arti that is most replacable.
	offloc := [4][5]int; //first [] is which char, second is is what type. stores which of the 5 artis in this slot are least good/most replacable
	offmap := [4][5][keep]float64; //stores the score of all the pieces
	//rollsperset := [maxdomain]float64;
	rollsperdomain := [(maxdomain+1)/2]float64; //what we need
	rpdc := [(maxdomain+1)/2]float64; //what we have
	rpdcpc := [4][(maxdomain+1)/2]float64; //rpdc per char
	done := [4]bool; //whether or not each char is done
	
	for _, c := range set {
		for _, p := range set[0] {
			rolls := 0
			for _, s := range desired[c] {
				rolls += Standardize(desired[c][s])/2
			}
			rollsperdomain[(p-1)/2] += rolls
		}
	}

	/*for _, v := range desired {
		if v > 0 {
			req += 1
		}
	}*/
	curdom := location of the max value of rollsperdomain//needtoimplementthis

NEXT:
	for count < maxTries && curdom!=-999 {
		count++
		var a Artifact
		a.Ok = true

		a.Set = curdom*2 + g.r.Intn(2)
		a.Slot = SlotType(g.r.Intn(5))
		a.Main, err = g.RandMain(a.Slot)
		if err != nil {
			return -1, err
		}
		//generate subs
		a.Subs = g.RandSubsNoHist(a.Main, 20)

		//update bag
		//onpieces, offpieces, onmin, onmap, offmin, offmap, rpdc, curdom = update(onpieces, offpieces, onmin, onmap, offmin, offmap, rpdc, a, main, desired)
		
		



		//uh just pretend this is a function
		for _, c := range onmin {
			updateRPDC := false; //optimially this is just a function but that initialization line would be so long
			ison := false; //lets us check less combinations
			
			
			if(!done[c]) {
				score := calcScore(a, c, main, desired)
				if isOnpiece(a, set) {
					if score > onmin[c][typetonum(a.SlotType)] {
						onpieces[c][typetonum(a.SlotType)][onloc[c][typetonum(a.SlotType)]] = a
						onmap[c][typetonum(a.SlotType)][onloc[c][typetonum(a.SlotType)]] = score
						onmin[c][typetonum(a.SlotType)], onloc[c][typetonum(a.SlotType)]] = calcMin(c, typetonum(a.SlotType), onmap)
						updateRPDC = true;
						ison = true;
					}
				} else {
					if score > offmin[c][typetonum(a.SlotType)] {
						offpieces[c][typetonum(a.SlotType)][offloc[c][typetonum(a.SlotType)]] = a
						offmap[c][typetonum(a.SlotType)][offloc[c][typetonum(a.SlotType)]] = score
						offmin[c][typetonum(a.SlotType)], offloc[c][typetonum(a.SlotType)]] = calcMin(c, typetonum(a.SlotType), offmap)
						updateRPDC = true;
					}
				}
			}
			
			if(updateRPDC) {
				
				
				
				curdom = findcurdom(rollsperdomain, rpdc);
			}
		}
		
	}
	
	if(count >= maxTries) {
		return -1, errors.New("maximum tries exceeded; requirement not met")
	}
	
	//once we're here we have all artifacts for ppl who need certain sets, farm more here if theres a char that can use full rainbow (2pc + rainbow should also already be complete)
	
	return count,nil;
}


func calcScore() {
	if(main[c][typetonum(a.SlotType)] != a.StatType) {
		return -1;
	}
	
	score := 0
	for _, s := range desired[c] {
		score += Standardize()*Standardize(desired[c][s]) //formula: #rolls of this stat on this arti * desired #rolls of this stat / ttl #desired rolls of all stats for this char. also standardizing it each time is probably a big ~~dps~~ speed loss so these should probably be stored
	}
}


























//func update(bag []Artifact, a Artifact, main [EndSlotType]StatType, desired [EndStatType]float64) ([]Artifact, float64) { as nice as it'd be to have an update function, i dont want to create this line lol
	//score should just be the total distance from desired stats, the lower it is
	//the better it is
	/*var prev, next, total float64 //score should be % of desired
	var replaced bool
	//if current slot is empty then just put it in
	if !bag[a.Slot].Ok {
		bag[a.Slot] = a
		replaced = true
	}

}*/

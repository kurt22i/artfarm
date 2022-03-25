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

type Artifact struct {
	Slot SlotType
	Main StatType
	Subs []float64
	Ok   bool
	Set  int
}

const maxTries = 100000000

//FarmArtifact return number of tries it tooks to reach the desired subs
//main is the desired main stat; if main == EndStatType then there's no requirement
func (g *Generator) FarmArtifact(main [][lib.EndSlotType]lib.StatType, desired [][lib.EndStatType]float64, set [][]int, maxdomain int ) (int, error) {
	var err error
	var req, score float64
	count := 0
	keep := 5 //the number of on and off pieces to store for optimizing per slot per char
	onpieces := [4][5][keep]Artifact //first [] is which char, second is what type it is, third is which of the 5 stored artis it is
	offpieces := [4][5][keep]Artifact
	onmin := [4][5]float64 //first [] is which char, second is is what type. stores the score the arti that is most replacable.
	onloc := [4][5]int //first [] is which char, second is is what type. stores which of the 5 artis in this slot are least good/most replacable
	onmap := [4][5][keep]float64 //stores the score of all the pieces
	offmin := [4][5]float64 //first [] is which char, second is is what type. stores the score the arti that is most replacable.
	offloc := [4][5]int //first [] is which char, second is is what type. stores which of the 5 artis in this slot are least good/most replacable
	offmap := [4][5][keep]float64 //stores the score of all the pieces
	rollsperdomain := [(maxdomain+1)/2]float64 //what we need
	rpdc := [(maxdomain+1)/2]float64 //what we have
	rpdcpc := [4][(maxdomain+1)/2]float64 //rpdc per char
	desrolls := [4][lib.EndStatType]float64 //desired standardized rolls
	ttldr := [4]float64 //total desired rolls for each char
	done := [4]bool //whether or not each char is done

	for _, c := range set {
		for _, p := range set[0] {
			rolls := 0
			for _, s := range desired[c] {
				rolls += Standardize(desired[c][s])/2
				desrolls[c][s] += Standardize(desired[c][s])/2
				ttldr[c] += Standardize(desired[c][s])/2
			}
			rollsperdomain[(p-1)/2] += rolls
		}
	}
	
	curdom := getDomain(rollsperdomain,rpdc); //location of the max value of rollsperdomain

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
			newloc :=-1;
			aslot := typetonum(a.SlotType)
			
			if(!done[c]) {
				score := calcScore(a, c, main, desired)
				if isOnpiece(a, set) {
					if score > onmin[c][aslot] {
						newloc = onloc[c][aslot]
						onpieces[c][aslot][newloc] = a
						onmap[c][aslot][newloc] = score
						onmin[c][aslot], onloc[c][aslot] = calcMin(c, aslot, onmap)
						updateRPDC = true
						ison = true
					}
				} else {
					if score > offmin[c][aslot] {
						newloc = offloc[c][aslot]
						offpieces[c][aslot][newloc] = a
						offmap[c][aslot][newloc] = score
						offmin[c][aslot], offloc[c][aslot] = calcMin(c, aslot, offmap)
						updateRPDC = true
					}
				}
			}
			
			if(updateRPDC) {
				//this is what should happen here:
				//-go thru all valid combinations (correct sets etc) from this char's onpieces and offpieces that involve the new artifact
				//-score each combination: foreachdesiredstat(score+= min(rolls of this stat with this combination, desired rolls of this stat))
				//-keep track of the combination with the highest score
				//-if a combination is found where score = ttl desired stat rolls (ie we found a set that works), change done[c] to true and exit loop (when done is changed to true, maybe these artifacts should be deleted, so that no other chars can use them, idk)
				//-once all combinations are searched, recalculate rpdcpc[c]
				//	-if the char uses 4pc set (or if two 2pc from same domain), this is simply done by rpdcpc[c][domainwiththatset] = score
				//  -if the char uses 2 (or 1&rainbow) 2pc sets, this is done by rpdcpc[c][domainwitha2pcset] = min(ttl desired rolls for this char/2 - 0.001, the score recalcuated from the winning combination but using only artifacts from this set) + 0.001 if done
				//recalculate rpdc which is just the sum of rpdcpc
				//recalculate curdom, which is the domain d where rollsperdomain[d]-rpdc[d] is the highest (when this is 0, set curdom to -999)
				
				maxcombo := [5]Artifact
				maxscore := 0
				combo := [5]Artifact
				combo[aslot] = a
				
				if ison {
					//1 off, 4 on.. ugh this doesnt support rainbow ;-; halp
					for _, i := range offpieces[c] {
						if i == a.SlotType { //dont check offpieces that are the same slot as our new onpiece
							continue
						}
						for _, j := range offpieces[c][i] { //ok now these are all the offpieces. search all combos with this offpiece, the new onpiece, and 3 other on pieces.
							combo[i] = offpieces[c][i][j]
							for k := 0; k<4; k++ { //slot of arti3... hm maybe should have array or something that takes 2 slots and returns the 3 other ones lol
								if k==i || k==aslot { //cant be same slot as existing artis
									continue
								}
								for l := k+1; l<4; l++ { //slot of arti4
									if l==i || l==aslot { //cant be same slot as existing artis
										continue
									}
									for m := l+1; m<4; m++ { //slot of arti5
										if m==i || m==aslot { //cant be same slot as existing artis
											continue
										}
										for _, n := range onpieces[c][k] { //each onpiece in slot k
											for _, o := range onpieces[c][l] { //each onpiece in slot l
												for _, p := range onpieces[c][m] { //each onpiece in slot m
													setcount := [maxdomain+1]int;
													setcount[a.Set]++;
													setcount[offpieces[c][i][j].Set]++;
													setcount[onpieces[c][k][n].Set]++;
													setcount[onpieces[c][l][o].Set]++;
													setcount[onpieces[c][m][p].Set]++;
													valid := true
													if set[c][0] == set[c][1] {
														if setcount[set[c][0]]<4 {//is this check needed? shouldnt be, since all onpieces should be of the right set in this case
															valid = false
														}
													} else {
														if(setcount[set[c][0]]<2 || setcount[set[c][1]]<2) {
															valid=false;
														}
													}
													if valid {
														combo[l]=onpieces[c][l][o]
														combo[k]=onpieces[c][k][n]
														combo[m]=onpieces[c][m][p]
														cscore := scoreCombo(combo, desrolls, set)
														if(cscore > maxscore) {
															maxscore = cscore;
															maxcombo = combo;
														}
													}
												}
											}
										}
									}
								}
							}
						}
					}
					
					
					
					
					
					
					//5 on
					for _, i := range onpieces[c] {
						if i == a.SlotType { //dont check onpieces that are the same slot as our new onpiece
							continue
						}
						for _, j := range onpieces[c][i] { //ok now this is lazy code because i only really needed this loop to go here for offpieces but its easier than restructuring the whole thing so
							combo[i] = onpieces[c][i][j]
							for k := 0; k<4; k++ { //slot of arti3... hm maybe should have array or something that takes 2 slots and returns the 3 other ones lol
								if k==i || k==aslot { //cant be same slot as existing artis
									continue
								}
								for l := k+1; l<4; l++ { //slot of arti4
									if l==i || l==aslot { //cant be same slot as existing artis
										continue
									}
									for m := l+1; m<4; m++ { //slot of arti5
										if m==i || m==aslot { //cant be same slot as existing artis
											continue
										}
										for _, n := range onpieces[c][k] { //each onpiece in slot k
											for _, o := range onpieces[c][l] { //each onpiece in slot l
												for _, p := range onpieces[c][m] { //each onpiece in slot m
													setcount := [maxdomain+1]int;
													setcount[a.Set]++;
													setcount[onpieces[c][i][j].Set]++;
													setcount[onpieces[c][k][n].Set]++;
													setcount[onpieces[c][l][o].Set]++;
													setcount[onpieces[c][m][p].Set]++;
													valid := true
													if set[c][0] != set[c][1] {
														if(setcount[set[c][0]]<2 || setcount[set[c][1]]<2) {
															valid=false;
														}
													}
													if valid {
														combo[l]=onpieces[c][l][o]
														combo[k]=onpieces[c][k][n]
														combo[m]=onpieces[c][m][p]
														cscore := scoreCombo(combo, desrolls, set)
														if(cscore > maxscore) {
															maxscore = cscore;
															maxcombo = combo;
														}
													}
												}
											}
										}
									}
								}
							}
						}
					}
					
				} else { //new artifact is offpiece. so, only need to search with this + 4 onpieces.
					for _, i := range onpieces[c] {
						if i == a.SlotType { //dont check onpieces that are the same slot as our new offpiece
							continue
						}
						for _, j := range onpieces[c][i] { //ok now this is lazy code because i only really needed this loop to go here for... uh idk im confused too at this point
							combo[i] = onpieces[c][i][j]
							for k := 0; k<4; k++ { //slot of arti3... hm maybe should have array or something that takes 2 slots and returns the 3 other ones lol
								if k==i || k==aslot { //cant be same slot as existing artis
									continue
								}
								for l := k+1; l<4; l++ { //slot of arti4
									if l==i || l==aslot { //cant be same slot as existing artis
										continue
									}
									for m := l+1; m<4; m++ { //slot of arti5
										if m==i || m==aslot { //cant be same slot as existing artis
											continue
										}
										for _, n := range onpieces[c][k] { //each onpiece in slot k
											for _, o := range onpieces[c][l] { //each onpiece in slot l
												for _, p := range onpieces[c][m] { //each onpiece in slot m
													setcount := [maxdomain+1]int;
													setcount[a.Set]++;
													setcount[onpieces[c][i][j].Set]++;
													setcount[onpieces[c][k][n].Set]++;
													setcount[onpieces[c][l][o].Set]++;
													setcount[onpieces[c][m][p].Set]++;
													valid := true
													if set[c][0] != set[c][1] {
														if(setcount[set[c][0]]<2 || setcount[set[c][1]]<2) {
															valid=false;
														}
													}
													if valid {
														combo[l]=onpieces[c][l][o]
														combo[k]=onpieces[c][k][n]
														combo[m]=onpieces[c][m][p]
														cscore := scoreCombo(combo, desrolls, set)
														if(cscore > maxscore) {
															maxscore = cscore;
															maxcombo = combo;
														}
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
				
				if(maxscore > ttldr[c] - 1.0/100000.0) { //if we're within a reasnoable margin of what we want (because float round errors lol), this char is done!
					done[c] = true
				}
				
				//ok now recalc rpdcpc
				if (set[c][0] == set[c][1] || (set[c][1]==set[c][0]+1 && set[c][1]%2==0)){ //if the char wants a 4pc set or 2 2pc from same domain, rpdcpc is just score. also this breaks if you enter the domains not in numerical order for a char so uh dont do that
					rpdcpc[c][(set[c][0]-1)/2] = maxscore;
				} else {
					if(!done[c]) {
						rpdcpc[c][(set[c][0]-1)/2] = Math.min(ttldr[c]/2 - 1.0/1000.0, scoreCombo(maxcombo, desrolls, set, c, 0);
						rpdcpc[c][(set[c][1]-1)/2] = Math.min(ttldr[c]/2 - 1.0/1000.0, scoreCombo(maxcombo, desrolls, set, c, 1);
					} else {
						rpdcpc[c][(set[c][0]-1)/2] = ttldr[c]/2
						rpdcpc[c][(set[c][1]-1)/2] = ttldr[c]/2
					}
				}
			
			
			
				curdom = getDomain(rollsperdomain, rpdc) //recalculate curdom, which is the domain d where rollsperdomain[d]-rpdc[d] is the highest (when this is 0, set curdom to -999)
			}
		}
		
	}
	
	if(count >= maxTries) {
		return -1, errors.New("maximum tries exceeded; requirement not met")
	}
	
	//once we're here we have all artifacts for ppl who need certain sets, farm more here if theres a char that can use full rainbow (2pc + rainbow should also already be complete.. actually no i think there's cases where it wouldn't be, in that case farm the 2pc domain bc why not)
	
	return count,nil
}


func calcScore() {
	if(main[c][typetonum(a.SlotType)] != a.StatType) {
		return -1
	}
	
	score := 0
	for _, s := range desired[c] {
		score += Standardize()*Standardize(desired[c][s]) //formula: #rolls of this stat on this arti * desired #rolls of this stat / ttl #desired rolls of all stats for this char. also standardizing it each time is probably a big ~~dps~~ speed loss so these should probably be stored
	}
}


func getDomain(rpd []float64, rpdc []float64) (float64){
	float64 max=0
	dom := -999
	for _, a := range rpd {
		if rpd[a]-rpdc[a]>max {
			max = rpd[a]-rpdc[a]
			dom = a
		}
	}
	return dom
}

func scoreCombo(combo []Artifact, desrolls [4][lib.EndStatType]float64, set [][]int, c int, selset int) (float64) {
	score := 0
	for _, s := range desrolls[c] {
		if(selset == -1) {
			score += Math.min(desrolls[c][s], Standardize(combo[0].Subs[s]+combo[1].Subs[s]+combo[2].Subs[s]+combo[3].Subs[s]+combo[4].Subs[s],s))
		} else if(desrolls[c][s] > 0){
			setscore := 0
			for _, a := range combo {
				if(combo[a].Set == set[c][selset]) {
					setscore += combo[a].Subs[s]
				}
			}
			score += Math.min(desrolls[c][s], Standardize(setscore))
		}
	}
	return score;
}
package translit

import (
	"fmt"
	"testing"
)

func TestString(t *testing.T) {
	s := String("Адская пасть Мохаве (Outwaters, The) 01 сезон. 09 серия    (Замена) 25,01.15", Short_SE_Form())
	fmt.Println(s)

}

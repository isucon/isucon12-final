using System;
using System.Buffers.Text;
using System.Collections.Generic;
using System.Text;
using UnityEngine;

namespace Data
{
    [Serializable]
    public class GachaMaster
    {
        public long id;
        public string name;
        public long startAt;
        public long endAt;
        public int displayOrder;
    }
    
    [Serializable]
    public class GachaItemMaster
    {
        public long id;
        public long gachaId;
        public int itemType;
        public long itemId;
        public int amount;
        public int weight;
    }
    
    [Serializable]
    public class ItemMaster
    {
        public long id;
        public int item_type;
        public string name;
        public string description;
        public int amount_per_sec;
        public int max_level;
        public int max_amount_perc_sec;
        public int base_exp_per_level;
        public int gained_exp;
        public int shortening_min;
    }

    public enum ItemType : int
    {
        Coin = 1,
        Hammer = 2,
        Exp = 3,
        Timer = 4,
    }
    
    public static class StaticItemMaster
    {
        public static Dictionary<long, ItemMaster> Items { get; } = new();

        static StaticItemMaster()
        {
            var decoded = Encoding.UTF8.GetString(Convert.FromBase64String(_raw));
            var items = JsonUtility.FromJson<ItemMasters>(decoded).items;
            foreach (var item in items)
            {
                Items.Add(item.id, item);
            }
        }

        [Serializable]
        private class ItemMasters
        {
            public ItemMaster[] items;
        }
        
        private static string _raw = @"eyJpdGVtcyI6W3siaWQiOjEsIml0ZW1fdHlwZSI6MSwibmFtZSI6IklTVS1DT0lOIiwiZGVzY3Jp
cHRpb24iOiJJU1UtQ09JTiIsImFtb3VudF9wZXJfc2VjIjoiIiwibWF4X2xldmVsIjoiIiwibWF4
X2Ftb3VudF9wZXJjX3NlYyI6IiIsImJhc2VfZXhwX3Blcl9sZXZlbCI6IiIsImdhaW5lZF9leHAi
OiIiLCJzaG9ydGVuaW5nX21pbiI6IiJ9LHsiaWQiOjIsIml0ZW1fdHlwZSI6MiwibmFtZSI6Iue0
meOBruODj+ODs+ODnuODvCIsImRlc2NyaXB0aW9uIjoi77yR56eS6ZaT44GrMeWAi+akheWtkOOC
kuS9nOOCjOOCi+ODj+ODs+ODnuODvCIsImFtb3VudF9wZXJfc2VjIjoxLCJtYXhfbGV2ZWwiOjUw
LCJtYXhfYW1vdW50X3BlcmNfc2VjIjo1MCwiYmFzZV9leHBfcGVyX2xldmVsIjoxMCwiZ2FpbmVk
X2V4cCI6IiIsInNob3J0ZW5pbmdfbWluIjoiIn0seyJpZCI6MywiaXRlbV90eXBlIjoyLCJuYW1l
Ijoi44K044Og44Gu44OP44Oz44Oe44O8IiwiZGVzY3JpcHRpb24iOiLvvJHnp5LplpPjgasy5YCL
5qSF5a2Q44KS5L2c44KM44KL44OP44Oz44Oe44O8IiwiYW1vdW50X3Blcl9zZWMiOjIsIm1heF9s
ZXZlbCI6NTAsIm1heF9hbW91bnRfcGVyY19zZWMiOjEwMCwiYmFzZV9leHBfcGVyX2xldmVsIjox
MCwiZ2FpbmVkX2V4cCI6IiIsInNob3J0ZW5pbmdfbWluIjoiIn0seyJpZCI6NCwiaXRlbV90eXBl
IjoyLCJuYW1lIjoi5pyo44Gu44OP44Oz44Oe44O8IiwiZGVzY3JpcHRpb24iOiLvvJHnp5LplpPj
gas15YCL5qSF5a2Q44KS5L2c44KM44KL44OP44Oz44Oe44O8IiwiYW1vdW50X3Blcl9zZWMiOjUs
Im1heF9sZXZlbCI6NTAsIm1heF9hbW91bnRfcGVyY19zZWMiOjI1MCwiYmFzZV9leHBfcGVyX2xl
dmVsIjoxMCwiZ2FpbmVkX2V4cCI6IiIsInNob3J0ZW5pbmdfbWluIjoiIn0seyJpZCI6NSwiaXRl
bV90eXBlIjoyLCJuYW1lIjoi44Ki44Or44Of44Gu44OP44Oz44Oe44O8IiwiZGVzY3JpcHRpb24i
OiLvvJHnp5LplpPjgasxMOWAi+akheWtkOOCkuS9nOOCjOOCi+ODj+ODs+ODnuODvCIsImFtb3Vu
dF9wZXJfc2VjIjoxMCwibWF4X2xldmVsIjo1MCwibWF4X2Ftb3VudF9wZXJjX3NlYyI6NTAwLCJi
YXNlX2V4cF9wZXJfbGV2ZWwiOjEwLCJnYWluZWRfZXhwIjoiIiwic2hvcnRlbmluZ19taW4iOiIi
fSx7ImlkIjo2LCJpdGVtX3R5cGUiOjIsIm5hbWUiOiLpiYTjga7jg4/jg7Pjg57jg7wiLCJkZXNj
cmlwdGlvbiI6Iu+8keenkumWk+OBqzIw5YCL5qSF5a2Q44KS5L2c44KM44KL44OP44Oz44Oe44O8
IiwiYW1vdW50X3Blcl9zZWMiOjIwLCJtYXhfbGV2ZWwiOjUwLCJtYXhfYW1vdW50X3BlcmNfc2Vj
IjoxMDAwLCJiYXNlX2V4cF9wZXJfbGV2ZWwiOjEwLCJnYWluZWRfZXhwIjoiIiwic2hvcnRlbmlu
Z19taW4iOiIifSx7ImlkIjo3LCJpdGVtX3R5cGUiOjIsIm5hbWUiOiLpnZLpioXjga7jg4/jg7Pj
g57jg7wiLCJkZXNjcmlwdGlvbiI6Iu+8keenkumWk+OBqzUw5YCL5qSF5a2Q44KS5L2c44KM44KL
44OP44Oz44Oe44O8IiwiYW1vdW50X3Blcl9zZWMiOjUwLCJtYXhfbGV2ZWwiOjUwLCJtYXhfYW1v
dW50X3BlcmNfc2VjIjoyNTAwLCJiYXNlX2V4cF9wZXJfbGV2ZWwiOjEwLCJnYWluZWRfZXhwIjoi
Iiwic2hvcnRlbmluZ19taW4iOiIifSx7ImlkIjo4LCJpdGVtX3R5cGUiOjIsIm5hbWUiOiLpi7zp
iYTjga7jg4/jg7Pjg57jg7wiLCJkZXNjcmlwdGlvbiI6Iu+8keenkumWk+OBqzEwMOWAi+akheWt
kOOCkuS9nOOCjOOCi+ODj+ODs+ODnuODvCIsImFtb3VudF9wZXJfc2VjIjoxMDAsIm1heF9sZXZl
bCI6ODAsIm1heF9hbW91bnRfcGVyY19zZWMiOjgwMDAsImJhc2VfZXhwX3Blcl9sZXZlbCI6MTEs
ImdhaW5lZF9leHAiOiIiLCJzaG9ydGVuaW5nX21pbiI6IiJ9LHsiaWQiOjksIml0ZW1fdHlwZSI6
MiwibmFtZSI6IumKheOBruODj+ODs+ODnuODvCIsImRlc2NyaXB0aW9uIjoi77yR56eS6ZaT44Gr
MjAw5YCL5qSF5a2Q44KS5L2c44KM44KL44OP44Oz44Oe44O8IiwiYW1vdW50X3Blcl9zZWMiOjIw
MCwibWF4X2xldmVsIjo4MCwibWF4X2Ftb3VudF9wZXJjX3NlYyI6MTYwMDAsImJhc2VfZXhwX3Bl
cl9sZXZlbCI6MTEsImdhaW5lZF9leHAiOiIiLCJzaG9ydGVuaW5nX21pbiI6IiJ9LHsiaWQiOjEw
LCJpdGVtX3R5cGUiOjIsIm5hbWUiOiLpioDjga7jg4/jg7Pjg57jg7wiLCJkZXNjcmlwdGlvbiI6
Iu+8keenkumWk+OBqzMwMOWAi+akheWtkOOCkuS9nOOCjOOCi+ODj+ODs+ODnuODvCIsImFtb3Vu
dF9wZXJfc2VjIjozMDAsIm1heF9sZXZlbCI6ODAsIm1heF9hbW91bnRfcGVyY19zZWMiOjI0MDAw
LCJiYXNlX2V4cF9wZXJfbGV2ZWwiOjExLCJnYWluZWRfZXhwIjoiIiwic2hvcnRlbmluZ19taW4i
OiIifSx7ImlkIjoxMSwiaXRlbV90eXBlIjoyLCJuYW1lIjoi54+K55Ga44Gu44OP44Oz44Oe44O8
IiwiZGVzY3JpcHRpb24iOiLvvJHnp5LplpPjgas1MDDlgIvmpIXlrZDjgpLkvZzjgozjgovjg4/j
g7Pjg57jg7wiLCJhbW91bnRfcGVyX3NlYyI6NTAwLCJtYXhfbGV2ZWwiOjgwLCJtYXhfYW1vdW50
X3BlcmNfc2VjIjo0MDAwMCwiYmFzZV9leHBfcGVyX2xldmVsIjoxMSwiZ2FpbmVkX2V4cCI6IiIs
InNob3J0ZW5pbmdfbWluIjoiIn0seyJpZCI6MTIsIml0ZW1fdHlwZSI6MiwibmFtZSI6IuODq+OD
k+ODvOOBruODj+ODs+ODnuODvCIsImRlc2NyaXB0aW9uIjoi77yR56eS6ZaT44GrMSwwMDDlgIvm
pIXlrZDjgpLkvZzjgozjgovjg4/jg7Pjg57jg7wiLCJhbW91bnRfcGVyX3NlYyI6MTAwMCwibWF4
X2xldmVsIjoxMDAsIm1heF9hbW91bnRfcGVyY19zZWMiOjEwMDAwMCwiYmFzZV9leHBfcGVyX2xl
dmVsIjoxMiwiZ2FpbmVkX2V4cCI6IiIsInNob3J0ZW5pbmdfbWluIjoiIn0seyJpZCI6MTMsIml0
ZW1fdHlwZSI6MiwibmFtZSI6IuOCteODleOCoeOCpOOCouOBruODj+ODs+ODnuODvCIsImRlc2Ny
aXB0aW9uIjoi77yR56eS6ZaT44GrNSwwMDDlgIvmpIXlrZDjgpLkvZzjgozjgovjg4/jg7Pjg57j
g7wiLCJhbW91bnRfcGVyX3NlYyI6NTAwMCwibWF4X2xldmVsIjoxMDAsIm1heF9hbW91bnRfcGVy
Y19zZWMiOjUwMDAwMCwiYmFzZV9leHBfcGVyX2xldmVsIjoxMiwiZ2FpbmVkX2V4cCI6IiIsInNo
b3J0ZW5pbmdfbWluIjoiIn0seyJpZCI6MTQsIml0ZW1fdHlwZSI6MiwibmFtZSI6IumHkeOBruOD
j+ODs+ODnuODvCIsImRlc2NyaXB0aW9uIjoi77yR56eS6ZaT44GrMTAsMDAw5YCL5qSF5a2Q44KS
5L2c44KM44KL44OP44Oz44Oe44O8IiwiYW1vdW50X3Blcl9zZWMiOjEwMDAwLCJtYXhfbGV2ZWwi
OjEwMCwibWF4X2Ftb3VudF9wZXJjX3NlYyI6MTAwMDAwMCwiYmFzZV9leHBfcGVyX2xldmVsIjox
MiwiZ2FpbmVkX2V4cCI6IiIsInNob3J0ZW5pbmdfbWluIjoiIn0seyJpZCI6MTUsIml0ZW1fdHlw
ZSI6MiwibmFtZSI6IuOCqOODoeODqeODq+ODieOBruODj+ODs+ODnuODvCIsImRlc2NyaXB0aW9u
Ijoi77yR56eS6ZaT44GrNTAsMDAw5YCL5qSF5a2Q44KS5L2c44KM44KL44OP44Oz44Oe44O8Iiwi
YW1vdW50X3Blcl9zZWMiOjUwMDAwLCJtYXhfbGV2ZWwiOjEwMCwibWF4X2Ftb3VudF9wZXJjX3Nl
YyI6NTAwMDAwMCwiYmFzZV9leHBfcGVyX2xldmVsIjoxMiwiZ2FpbmVkX2V4cCI6IiIsInNob3J0
ZW5pbmdfbWluIjoiIn0seyJpZCI6MTYsIml0ZW1fdHlwZSI6MiwibmFtZSI6IuODgOOCpOOCouOD
ouODs+ODieOBruODj+ODs+ODnuODvCIsImRlc2NyaXB0aW9uIjoi77yR56eS6ZaT44GrMTAwLDAw
MOWAi+akheWtkOOCkuS9nOOCjOOCi+ODj+ODs+ODnuODvCIsImFtb3VudF9wZXJfc2VjIjoxMDAw
MDAsIm1heF9sZXZlbCI6MTAwLCJtYXhfYW1vdW50X3BlcmNfc2VjIjoxMDAwMDAwMCwiYmFzZV9l
eHBfcGVyX2xldmVsIjoxMiwiZ2FpbmVkX2V4cCI6IiIsInNob3J0ZW5pbmdfbWluIjoiIn0seyJp
ZCI6MTcsIml0ZW1fdHlwZSI6MywibmFtZSI6IuW8t+WMlue0oOadkOWwjyIsImRlc2NyaXB0aW9u
Ijoi57WM6aiT5YCkMTDnjbLlvpciLCJhbW91bnRfcGVyX3NlYyI6IiIsIm1heF9sZXZlbCI6IiIs
Im1heF9hbW91bnRfcGVyY19zZWMiOiIiLCJiYXNlX2V4cF9wZXJfbGV2ZWwiOiIiLCJnYWluZWRf
ZXhwIjoxMCwic2hvcnRlbmluZ19taW4iOiIifSx7ImlkIjoxOCwiaXRlbV90eXBlIjozLCJuYW1l
Ijoi5by35YyW57Sg5p2Q5LitIiwiZGVzY3JpcHRpb24iOiLntYzpqJPlgKQxMDDnjbLlvpciLCJh
bW91bnRfcGVyX3NlYyI6IiIsIm1heF9sZXZlbCI6IiIsIm1heF9hbW91bnRfcGVyY19zZWMiOiIi
LCJiYXNlX2V4cF9wZXJfbGV2ZWwiOiIiLCJnYWluZWRfZXhwIjoxMDAsInNob3J0ZW5pbmdfbWlu
IjoiIn0seyJpZCI6MTksIml0ZW1fdHlwZSI6MywibmFtZSI6IuW8t+WMlue0oOadkOWkpyIsImRl
c2NyaXB0aW9uIjoi57WM6aiT5YCkMSwwMDDnjbLlvpciLCJhbW91bnRfcGVyX3NlYyI6IiIsIm1h
eF9sZXZlbCI6IiIsIm1heF9hbW91bnRfcGVyY19zZWMiOiIiLCJiYXNlX2V4cF9wZXJfbGV2ZWwi
OiIiLCJnYWluZWRfZXhwIjoxMDAwLCJzaG9ydGVuaW5nX21pbiI6IiJ9LHsiaWQiOjIwLCJpdGVt
X3R5cGUiOjMsIm5hbWUiOiLlvLfljJbntKDmnZDnibnlpKciLCJkZXNjcmlwdGlvbiI6Iue1jOmo
k+WApDEwLDAwMOeNsuW+lyIsImFtb3VudF9wZXJfc2VjIjoiIiwibWF4X2xldmVsIjoiIiwibWF4
X2Ftb3VudF9wZXJjX3NlYyI6IiIsImJhc2VfZXhwX3Blcl9sZXZlbCI6IiIsImdhaW5lZF9leHAi
OjEwMDAwLCJzaG9ydGVuaW5nX21pbiI6IiJ9LHsiaWQiOjIxLCJpdGVtX3R5cGUiOjQsIm5hbWUi
OiIx5YiG55+t57iu44K/44Kk44Oe44O8IiwiZGVzY3JpcHRpb24iOiIx5YiG55+t57iuIiwiYW1v
dW50X3Blcl9zZWMiOiIiLCJtYXhfbGV2ZWwiOiIiLCJtYXhfYW1vdW50X3BlcmNfc2VjIjoiIiwi
YmFzZV9leHBfcGVyX2xldmVsIjoiIiwiZ2FpbmVkX2V4cCI6IiIsInNob3J0ZW5pbmdfbWluIjox
fSx7ImlkIjoyMiwiaXRlbV90eXBlIjo0LCJuYW1lIjoiM+WIhuefree4ruOCv+OCpOODnuODvCIs
ImRlc2NyaXB0aW9uIjoiM+WIhuefree4riIsImFtb3VudF9wZXJfc2VjIjoiIiwibWF4X2xldmVs
IjoiIiwibWF4X2Ftb3VudF9wZXJjX3NlYyI6IiIsImJhc2VfZXhwX3Blcl9sZXZlbCI6IiIsImdh
aW5lZF9leHAiOiIiLCJzaG9ydGVuaW5nX21pbiI6M30seyJpZCI6MjMsIml0ZW1fdHlwZSI6NCwi
bmFtZSI6IjXliIbnn63nuK7jgr/jgqTjg57jg7wiLCJkZXNjcmlwdGlvbiI6IjXliIbnn63nuK4i
LCJhbW91bnRfcGVyX3NlYyI6IiIsIm1heF9sZXZlbCI6IiIsIm1heF9hbW91bnRfcGVyY19zZWMi
OiIiLCJiYXNlX2V4cF9wZXJfbGV2ZWwiOiIiLCJnYWluZWRfZXhwIjoiIiwic2hvcnRlbmluZ19t
aW4iOjV9LHsiaWQiOjI0LCJpdGVtX3R5cGUiOjQsIm5hbWUiOiIxMOWIhuefree4ruOCv+OCpOOD
nuODvCIsImRlc2NyaXB0aW9uIjoiMTDliIbnn63nuK4iLCJhbW91bnRfcGVyX3NlYyI6IiIsIm1h
eF9sZXZlbCI6IiIsIm1heF9hbW91bnRfcGVyY19zZWMiOiIiLCJiYXNlX2V4cF9wZXJfbGV2ZWwi
OiIiLCJnYWluZWRfZXhwIjoiIiwic2hvcnRlbmluZ19taW4iOjEwfSx7ImlkIjoyNSwiaXRlbV90
eXBlIjo0LCJuYW1lIjoiMzDliIbnn63nuK7jgr/jgqTjg57jg7wiLCJkZXNjcmlwdGlvbiI6IjMw
5YiG55+t57iuIiwiYW1vdW50X3Blcl9zZWMiOiIiLCJtYXhfbGV2ZWwiOiIiLCJtYXhfYW1vdW50
X3BlcmNfc2VjIjoiIiwiYmFzZV9leHBfcGVyX2xldmVsIjoiIiwiZ2FpbmVkX2V4cCI6IiIsInNo
b3J0ZW5pbmdfbWluIjozMH0seyJpZCI6MjYsIml0ZW1fdHlwZSI6NCwibmFtZSI6IjYw5YiG55+t
57iu44K/44Kk44Oe44O8IiwiZGVzY3JpcHRpb24iOiI2MOWIhuefree4riIsImFtb3VudF9wZXJf
c2VjIjoiIiwibWF4X2xldmVsIjoiIiwibWF4X2Ftb3VudF9wZXJjX3NlYyI6IiIsImJhc2VfZXhw
X3Blcl9sZXZlbCI6IiIsImdhaW5lZF9leHAiOiIiLCJzaG9ydGVuaW5nX21pbiI6NjB9LHsiaWQi
OjI3LCJpdGVtX3R5cGUiOjQsIm5hbWUiOiIxMjDliIbnn63nuK7jgr/jgqTjg57jg7wiLCJkZXNj
cmlwdGlvbiI6IjEyMOWIhuefree4riIsImFtb3VudF9wZXJfc2VjIjoiIiwibWF4X2xldmVsIjoi
IiwibWF4X2Ftb3VudF9wZXJjX3NlYyI6IiIsImJhc2VfZXhwX3Blcl9sZXZlbCI6IiIsImdhaW5l
ZF9leHAiOiIiLCJzaG9ydGVuaW5nX21pbiI6MTIwfSx7ImlkIjoyOCwiaXRlbV90eXBlIjo0LCJu
YW1lIjoiMzAw5YiG55+t57iu44K/44Kk44Oe44O8IiwiZGVzY3JpcHRpb24iOiIzMDDliIbnn63n
uK4iLCJhbW91bnRfcGVyX3NlYyI6IiIsIm1heF9sZXZlbCI6IiIsIm1heF9hbW91bnRfcGVyY19z
ZWMiOiIiLCJiYXNlX2V4cF9wZXJfbGV2ZWwiOiIiLCJnYWluZWRfZXhwIjoiIiwic2hvcnRlbmlu
Z19taW4iOjMwMH0seyJpZCI6MjksIml0ZW1fdHlwZSI6NCwibmFtZSI6IjYwMOWIhuefree4ruOC
v+OCpOODnuODvCIsImRlc2NyaXB0aW9uIjoiNjAw5YiG55+t57iuIiwiYW1vdW50X3Blcl9zZWMi
OiIiLCJtYXhfbGV2ZWwiOiIiLCJtYXhfYW1vdW50X3BlcmNfc2VjIjoiIiwiYmFzZV9leHBfcGVy
X2xldmVsIjoiIiwiZ2FpbmVkX2V4cCI6IiIsInNob3J0ZW5pbmdfbWluIjo2MDB9LHsiaWQiOjMw
LCJpdGVtX3R5cGUiOjQsIm5hbWUiOiIxNDQw5YiG55+t57iu44K/44Kk44Oe44O8IiwiZGVzY3Jp
cHRpb24iOiIxLDQ0MOWIhuefree4riIsImFtb3VudF9wZXJfc2VjIjoiIiwibWF4X2xldmVsIjoi
IiwibWF4X2Ftb3VudF9wZXJjX3NlYyI6IiIsImJhc2VfZXhwX3Blcl9sZXZlbCI6IiIsImdhaW5l
ZF9leHAiOiIiLCJzaG9ydGVuaW5nX21pbiI6MTQ0MH1dfQ==";
    }
}

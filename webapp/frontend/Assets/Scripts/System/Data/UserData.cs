using System;
using System.Collections.Generic;

namespace Data
{
    [Serializable]
    public class User
    {
        public long id;
        public long isuCoin;
        
        public string viewerId;
        public string sessionId;

        public long lastGetRewardAt;
        public long lastActivatedAt;
    }

    [Serializable]
    public class UserDevice
    {
        public long id;
        public long userId;
        public string platformId;
        public int platformType;
    }

    [Serializable]
    public class UserCard
    {
        public long id;
        public long userId;
        public long cardId;
        public int amountPerSec;
        public int level;
        public long totalExp;
    }

    [Serializable]
    public class UserDeck
    {
        public long id;
        public long userId;
        public long cardId1;
        public long cardId2;
        public long cardId3;
    }
    
    [Serializable]
    public class UserDeckData
    {
        public long id;
        public long userId;
        public UserCard cardId1;
        public UserCard cardId2;
        public UserCard cardId3;
    }

    [Serializable]
    public class UserItem
    {
        public long id;
        public long userId;
        public int itemType;
        public long itemId;
        public int amount;
    }

    [Serializable]
    public class UserLoginBonus
    {
        public long id;
        public long userId;
        public long loginBonuseId;
        public int lastRewardSequence;
        public int loopCount;
    }

    [Serializable]
    public class UserPresent
    {
        public long id;
        public long userId;
        public string sentAt;
        public int itemType;
        public long itemId;
        public int amount;
        public string presentMessage;
    }
    

    public class UserData
    {
        public class IsuCoin
        {
            public int totalPerSec;
            public long pastTime;
            public long refreshTime;
        }

        public class Deck
        {
            public UserCard card1;
            public UserCard card2;
            public UserCard card3;
        }
        
        public User user = new();
        public UserDeck userDeck;
        public List<UserLoginBonus> loginBonuses = new();
        public List<UserPresent> presents = new();

        public IsuCoin isuCoin = new();
        public Deck deck = new();
    }
}

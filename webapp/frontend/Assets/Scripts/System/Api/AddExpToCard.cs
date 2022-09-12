using System;
using System.Threading.Tasks;
using Data;

namespace Network
{
    [Serializable]
    public class AddExpToCardRequest : CommonRequest
    {
        [Serializable]
        public class Item
        {
            public long id;
            public int amount;

            public Item(long id, int amount)
            {
                this.id = id;
                this.amount = amount;
            }
        }
        
        public string oneTimeToken;
        public Item[] items;
    }

    [Serializable]
    public class AddExpToCardResponse : CommonResponse
    {
    }
    
    public class AddExpToCardApi : ApiBase<AddExpToCardRequest, AddExpToCardResponse>
    {
        public override string Path => $"/user/{GameManager.userData.user.id}/card/addexp/{_cardId}";

        private long _cardId;

        public AddExpToCardApi(AddExpToCardRequest req, long cardId)
        {
            RequestData = req;
            _cardId = cardId;
        }
    }

    public partial class ApiClient
    {
        public async Task<AddExpToCardResponse> AddExpToCardAsync(long cardId, AddExpToCardRequest.Item[] items)
        {
            var req = new AddExpToCardRequest
            {
                viewerId = this.ViewerId,
                oneTimeToken = this._oneTimeToken,
                items = items,
            };
            var api = new AddExpToCardApi(req, cardId);
            var res = await Post(api);
            return res;
        }
    }
}
